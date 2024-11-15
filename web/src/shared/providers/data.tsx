import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { type Client, RequestEventAction } from '~/api'
import { Database } from '~/db'
import { UsedStorageKeys, useStorage } from '~/shared'

export type Session = {
  sID: string
  responseCode: number
  responseHeaders: Array<{ name: string; value: string }>
  responseDelay: number
  responseBody: Uint8Array
}

export type Request = {
  rID: string
  clientAddress: string // IPv4 or IPv6
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS' | 'CONNECT' | 'TRACE' | string
  headers: Array<{ name: string; value: string }>
  url: URL
  get payload(): Promise<Uint8Array | null> | null // the payload is lazy-loaded to avoid memory overuse
  capturedAt: Date
}

type SessionEvents = {
  onNewRequest: (r: Omit<Request, 'payload'>) => void // server does not send the payload
  onRequestDelete: (r: Omit<Request, 'payload'>) => void // server does not send the payload
  onRequestsClear: () => void
}

type DataContext = {
  /** The last used session ID (updates every time a session is switched) */
  readonly lastUsedSID: string | null

  /** Create a new session */
  newSession({
    statusCode,
    headers,
    delay,
    responseBody,
  }: {
    statusCode?: number
    headers?: Record<string, string>
    delay?: number
    responseBody?: Uint8Array
  }): Promise<Readonly<Session>>

  /**
   * Switch to a session with the given ID.
   *
   * NOTE: The first promise resolves when the session and requests are loaded from the database (FAST), and the
   * second one resolves when the session and requests are loaded from the server (SLOW).
   */
  switchToSession(sID: string, listeners?: Partial<SessionEvents>): Promise<() => Promise<void>>

  /** Current active session */
  readonly session: Readonly<Session> | null

  /** The list of all session IDs, available to the user */
  readonly allSessionIDs: ReadonlyArray<string>

  /**
   * Destroy a session with the given ID.
   *
   * NOTE: The first promise resolves when the session is removed from the database (FAST), and the second one
   * resolves when the session is removed from the server (SLOW).
   */
  destroySession(sID: string): Promise<() => Promise<void>>

  /** Current active request */
  readonly request: Readonly<Request> | null

  /** The list of requests for the current session, ordered by the captured time (from newest to oldest) */
  readonly requests: ReadonlyArray<Request>

  /**
   * Switch to a request with the given session and request ID.
   *
   * NOTE: The first promise resolves when the request is loaded from the database (FAST), and the second one
   * resolves when the request is loaded from the server (SLOW).
   */
  switchToRequest(sID: string, rID: string | null): Promise<() => Promise<void>>

  /**
   * Remove a request with the given session and request ID.
   *
   * NOTE: The first promise resolves when the request is removed from the database (FAST), and the second one
   * resolves when the request is removed from the server (SLOW).
   */
  removeRequest(sID: string, rID: string): Promise<() => Promise<void>>

  /**
   * Remove all requests for the session with the given ID.
   *
   * NOTE: The first promise resolves when the requests are removed from the database (FAST), and the second one
   * resolves when the requests are removed from the server (SLOW).
   */
  removeAllRequests(sID: string): Promise<() => Promise<void>>

  /** Limit the number of requests by removing the oldest ones, if the count exceeds the limit */
  setRequestsCount(limit: number): void

  /** The URL for the webhook (if session is active) */
  readonly webHookUrl: Readonly<URL> | null
}

const notInitialized = (): never => {
  throw new Error('DataProvider is not initialized')
}

const dataContext = createContext<DataContext>({
  lastUsedSID: null,
  newSession: () => notInitialized(),
  switchToSession: () => notInitialized(),
  session: null,
  allSessionIDs: [],
  destroySession: () => notInitialized(),
  request: null,
  requests: [],
  switchToRequest: () => notInitialized(),
  removeRequest: () => notInitialized(),
  removeAllRequests: () => notInitialized(),
  setRequestsCount: () => notInitialized(),
  webHookUrl: null,
})

/** Sort requests by the captured time (from newest to oldest) */
const requestsSorter = <T extends { capturedAt: Date }>(a: T, b: T) => b.capturedAt.getTime() - a.capturedAt.getTime()

/** Helper function to get the request payload from the database (lazy-loaded) */
const payloadGetter = (db: Database, rID: string): { payload: Request['payload'] } => {
  return {
    get payload() {
      return new Promise<Uint8Array | null>((resolve, reject) => {
        db.getRequest(rID)
          .then((r) => {
            if (r) {
              resolve(r.payload)
            } else {
              reject(new Error('Request not found in the database'))
            }
          })
          .catch(reject)
      })
    },
  }
}

/**
 * DataProvider is a context provider that manages application data.
 *
 * Think of it as the **core** of the business logic, handling all data and key methods related to sessions and requests.
 */
export const DataProvider: React.FC<{
  api: Client
  db: Database
  errHandler: (err: Error | unknown) => void // error handler for non-critical errors
  children: React.JSX.Element
}> = ({ api, db, errHandler, children }) => {
  const [lastUsedSID, setLastUsedSID] = useStorage<string | null>(null, UsedStorageKeys.SessionsLastUsed, 'local')
  const [session, setSession] = useState<Readonly<Session> | null>(null)
  const [allSessionIDs, setAllSessionIDs] = useState<ReadonlyArray<string>>([])
  const [request, setRequest] = useState<Readonly<Request> | null>(null)
  const [requests, setRequests] = useState<ReadonlyArray<Request>>([])
  const [webHookUrl, setWebHookUrl] = useState<URL | null>(null)

  // the subscription closer function (if not null, it means the subscription is active)
  const closeSubRef = useRef<(() => void) | null>(null)

  /** Subscribe to the session requests on the server */
  const subscribeToRequestEvents = useCallback(
    (sID: string, listeners?: Partial<SessionEvents>) => {
      return new Promise<void>((resolve, reject) => {
        // unsubscribe from the previous session requests
        if (closeSubRef.current) {
          closeSubRef.current()
        }

        closeSubRef.current = null

        // subscribe to the session requests on the server
        api
          .subscribeToSessionRequests(sID, {
            onUpdate: (requestEvent): void => {
              switch (requestEvent.action) {
                // a new request was captured
                case RequestEventAction.create: {
                  const req = requestEvent.request

                  if (req) {
                    // save the request to the database (without payload)
                    db.createRequest({
                      sID: sID,
                      rID: req.uuid,
                      method: req.method,
                      clientAddress: req.clientAddress,
                      url: req.url.toString(),
                      payload: null, // server does not send the payload
                      capturedAt: req.capturedAt,
                      headers: [...req.headers],
                    })
                      .then(() => {
                        // append the new request in front of the list (update the state)
                        setRequests((prev) => [
                          Object.freeze({
                            ...payloadGetter(db, req.uuid),
                            rID: req.uuid,
                            clientAddress: req.clientAddress,
                            method: req.method,
                            headers: [...req.headers],
                            url: req.url,
                            capturedAt: req.capturedAt,
                          }),
                          ...prev,
                        ])

                        // invoke the listener callback
                        listeners?.onNewRequest?.(
                          Object.freeze({
                            rID: req.uuid,
                            clientAddress: req.clientAddress,
                            method: req.method,
                            headers: [...req.headers],
                            url: req.url,
                            capturedAt: req.capturedAt,
                          })
                        )

                        // get the request payload from the server
                        api
                          .getSessionRequest(sID, req.uuid)
                          .then((req) => {
                            // update the request in the database with the payload
                            db.createRequest({
                              sID: sID,
                              rID: req.uuid,
                              method: req.method,
                              clientAddress: req.clientAddress,
                              url: req.url.toString(),
                              payload: req.requestPayload,
                              capturedAt: req.capturedAt,
                              headers: [...req.headers],
                            }).catch(errHandler)
                          })
                          .catch(errHandler)
                      })
                      .catch(errHandler)
                  }

                  break
                }

                // a request was deleted
                case RequestEventAction.delete: {
                  const req = requestEvent.request

                  if (req) {
                    // remove the request from the list
                    setRequests((prev) => prev.filter((r) => r.rID !== req.uuid))

                    // invoke the listener callback
                    listeners?.onRequestDelete?.({
                      rID: req.uuid,
                      clientAddress: req.clientAddress,
                      method: req.method,
                      headers: [...req.headers],
                      url: req.url,
                      capturedAt: req.capturedAt,
                    })

                    // remove the request from the database
                    db.deleteRequest(req.uuid).catch(errHandler)
                  }

                  break
                }

                // all requests were cleared
                case RequestEventAction.clear: {
                  // clear the requests list
                  setRequests([])

                  // invoke the listener callback
                  listeners?.onRequestsClear?.()

                  // clear the requests from the database
                  db.deleteAllRequests(sID).catch(errHandler)

                  break
                }
              }
            },
            onError: (err) => reject(err),
          })
          .then((closer) => {
            closeSubRef.current = closer
            resolve()
          })
          .catch(reject)
      })
    },
    [api, db, errHandler]
  )

  /** Create a new session */
  const newSession = useCallback(
    ({
      statusCode = 200, // default session options
      headers = {},
      delay = 0,
      responseBody = new Uint8Array(),
    }: {
      statusCode?: number
      headers?: Record<string, string>
      delay?: number
      responseBody?: Uint8Array
    }): Promise<Readonly<Session>> => {
      return new Promise((resolve, reject) => {
        // save the session to the server
        api
          .newSession({ statusCode, headers, delay, responseBody })
          .then((opts) => {
            // add the session ID to the list of all session IDs (update the state)
            setAllSessionIDs((prev) => [...prev, opts.uuid])

            // save the session to the database
            db.createSession({
              sID: opts.uuid,
              responseCode: statusCode,
              responseDelay: delay,
              responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
              responseBody,
              createdAt: opts.createdAt,
            })
              .then(() => {
                resolve(
                  Object.freeze({
                    sID: opts.uuid,
                    responseCode: statusCode,
                    responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
                    responseDelay: delay,
                    responseBody,
                  })
                )
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  /**
   * Load the requests for the session with the given ID.
   *
   * This action will reset the requests list and update it with the new data.
   *
   * NOTE: The first promise resolves when the requests are loaded from the database (FAST), and the second one
   * resolves when the requests are loaded from the server (SLOW).
   */
  const loadRequests = useCallback(
    (sID: string): Promise<() => Promise<void>> => {
      return new Promise((resolveFast, rejectFast) => {
        // load requests for the session from the database (fast)
        db.getSessionRequests(sID)
          .then((dbList) =>
            // update the requests list (first state update, to show the data from the database)
            setRequests(
              dbList
                .map((r) =>
                  Object.freeze({
                    ...payloadGetter(db, r.rID),
                    rID: r.rID,
                    clientAddress: r.clientAddress,
                    method: r.method,
                    headers: [...r.headers],
                    url: new URL(r.url),
                    capturedAt: r.capturedAt,
                  })
                )
                .sort(requestsSorter)
            )
          )
          // load requests from the server (slow)
          .then(() =>
            resolveFast(
              () =>
                new Promise<void>((resolveSlow, rejectSlow) => {
                  api
                    .getSessionRequests(sID)
                    .then((reqs) => {
                      // update the requests list (second state update, to show the fresh data)
                      setRequests(
                        reqs
                          .map((r) => ({
                            ...payloadGetter(db, r.uuid),
                            rID: r.uuid,
                            clientAddress: r.clientAddress,
                            method: r.method,
                            headers: [...r.headers],
                            url: r.url,
                            capturedAt: r.capturedAt,
                          }))
                          .sort(requestsSorter)
                      )

                      // update the requests in the database (for the future use)
                      db.createRequest(
                        ...reqs.map((r) => ({
                          sID: sID,
                          rID: r.uuid,
                          method: r.method,
                          clientAddress: r.clientAddress,
                          url: r.url.toString(),
                          capturedAt: r.capturedAt,
                          headers: [...r.headers],
                          payload: r.requestPayload,
                        }))
                      )
                        .then(resolveSlow)
                        .catch(rejectSlow)
                    })
                    .catch(rejectSlow)
                })
            )
          )
          .catch(rejectFast)
      })
    },
    [db, api]
  )

  /**
   * Switch to a session with the given ID.
   *
   * NOTE: The first promise resolves when the session and requests are loaded from the database (FAST), and the
   * second one resolves when the session and requests are loaded from the server (SLOW).
   */
  const switchToSession = useCallback(
    (sID: string, listeners?: Partial<SessionEvents>) => {
      return new Promise<() => Promise<void>>((resolveFast, rejectFast) => {
        // first, try to find out if the session exists in the database
        db.getSession(sID)
          .then((dbSession) => {
            // if the session exists in the database
            if (dbSession) {
              // set the session as the current session (update the state)
              setSession({
                sID: dbSession.sID,
                responseCode: dbSession.responseCode,
                responseDelay: dbSession.responseDelay,
                responseHeaders: dbSession.responseHeaders,
                responseBody: dbSession.responseBody,
              })

              // update the last used session ID
              setLastUsedSID(dbSession.sID)

              // load the requests for the session
              loadRequests(dbSession.sID)
                // when the requests are loaded from the database
                .then((requestsSlow) => {
                  // resolve the first (fast) promise
                  resolveFast(
                    // and return the second (slow) promise
                    () =>
                      new Promise<void>((resolveSlow, rejectSlow) => {
                        // load the requests from the server
                        requestsSlow()
                          // when requests are loaded from the server, subscribe to the session requests
                          .then(() => subscribeToRequestEvents(dbSession.sID, listeners))
                          // and resolve the second (slow) promise
                          .then(resolveSlow)
                          // on error loading the requests from the server
                          .catch((err) => {
                            setLastUsedSID(null) // unset the last used session ID
                            rejectSlow(err) // reject the second (slow) promise
                          })
                      })
                  )
                })
                // on requests load error (from the database)
                .catch((err) => {
                  setLastUsedSID(null) // unset the last used session ID
                  rejectFast(err) // reject the first (fast) promise
                })
            } else {
              // otherwise, try to get it from the server (since we need to load from the server, we should resolve
              // the first (fast) promise with the second one (slow))
              resolveFast(
                () =>
                  new Promise<void>((resolveSlow, rejectSlow) => {
                    // load the session from the server
                    api
                      .getSession(sID)
                      .then((apiSession) => {
                        // save the session to the database
                        db.createSession({
                          sID: apiSession.uuid,
                          responseCode: apiSession.response.statusCode,
                          responseDelay: apiSession.response.delay,
                          responseHeaders: [...apiSession.response.headers],
                          responseBody: apiSession.response.body,
                          createdAt: apiSession.createdAt,
                        })
                          // when the session is saved to the database
                          .then(() => {
                            // add the session ID to the list of all session IDs (at the end)
                            setAllSessionIDs((prev) => [...prev, apiSession.uuid])

                            // set the session as the current session (update the state)
                            setSession({
                              sID: apiSession.uuid,
                              responseCode: apiSession.response.statusCode,
                              responseDelay: apiSession.response.delay,
                              responseHeaders: [...apiSession.response.headers],
                              responseBody: apiSession.response.body,
                            })

                            // update the last used session ID
                            setLastUsedSID(apiSession.uuid)

                            // and load the requests for the session
                            loadRequests(apiSession.uuid)
                              // when the requests are loaded from the database
                              .then((requestsSlow) => {
                                // load the requests from the server
                                requestsSlow()
                                  // when requests are loaded from the server, subscribe to the session requests
                                  .then(() => subscribeToRequestEvents(apiSession.uuid, listeners))
                                  // and resolve the second (slow) promise
                                  .then(resolveSlow)
                                  // on error loading the requests from the server
                                  .catch((err) => {
                                    setLastUsedSID(null) // unset the last used session ID
                                    rejectSlow(err) // reject the second (slow) promise
                                  })
                              })
                              // on error loading the requests from the database
                              .catch((err) => {
                                setLastUsedSID(null) // unset the last used session ID
                                rejectSlow(err) // reject the second (slow) promise
                              })
                          })
                          // on error saving the session to the database
                          .catch((err) => {
                            setLastUsedSID(null) // unset the last used session ID
                            rejectSlow(err) // reject the second (slow) promise
                          })
                      })
                      // on error loading the session from the server
                      .catch((err) => {
                        setLastUsedSID(null) // unset the last used session ID
                        rejectSlow(err) // reject the second (slow) promise
                      })
                  })
              )
            }
          })
          .catch((err) => {
            setLastUsedSID(null)
            rejectFast(err)
          })
      })
    },
    [api, db, loadRequests, subscribeToRequestEvents, setLastUsedSID]
  )

  /**
   * Destroy a session with the given ID.
   *
   * NOTE: The first promise resolves when the session is removed from the database (FAST), and the second one
   * resolves when the session is removed from the server (SLOW).
   */
  const destroySession = useCallback(
    (sID: string): Promise<() => Promise<void>> => {
      return new Promise((resolveFast, rejectFast) => {
        // remove the session from the database first (fast)
        db.deleteSession(sID)
          .then(() => {
            // remove the session from the list of all session IDs (update the state)
            setAllSessionIDs((prev) => prev.filter((id) => id !== sID))

            // remove the session from the server (slow)
            resolveFast(
              () =>
                new Promise<void>((resolveSlow, rejectSlow) => {
                  // remove from the server too
                  api
                    .deleteSession(sID)
                    .then((ok) => {
                      if (ok) {
                        resolveSlow()
                      } else {
                        rejectSlow(new Error('Failed to delete the session on the server'))
                      }
                    })
                    .catch(rejectSlow)
                })
            )
          })
          .catch(rejectFast)
      })
    },
    [api, db]
  )

  /**
   * Switch to a request with the given session and request ID.
   *
   * NOTE: The first promise resolves when the request is loaded from the database (FAST), and the second one
   * resolves when the request is loaded from the server (SLOW).
   */
  const switchToRequest2 = useCallback(
    (sID: string, rID: string | null): Promise<() => Promise<void>> => {
      if (!rID) {
        setRequest(null)

        return new Promise(() => Promise.resolve())
      }
      // TODO: remove request from the database if API returns 404
      return new Promise<() => Promise<void>>((resolveFast, rejectFast) => {
        // get the request from the database (fast)
        db.getRequest(rID)
          .then((req) => {
            // if the request exists in the database
            if (req) {
              // set the current request with the data from the database
              setRequest({
                rID: req.rID,
                clientAddress: req.clientAddress,
                method: req.method,
                headers: [...req.headers],
                url: new URL(req.url),
                capturedAt: req.capturedAt,
                get payload() {
                  return Promise.resolve(req.payload)
                },
              })

              // is the payload already here, just resolve the first (fast) promise with the empty second one
              if (req.payload !== null) {
                resolveFast(() => Promise.resolve())
              } else {
                // if the payload is not loaded yet, try to get it from the server (slow)
                resolveFast(
                  () =>
                    new Promise<void>((resolveSlow, rejectSlow) => {
                      api
                        .getSessionRequest(sID, rID)
                        .then((req) => {
                          // update the current request state with the payload
                          setRequest((prev) => {
                            if (prev) {
                              return {
                                rID: rID,
                                clientAddress: req.clientAddress,
                                method: req.method,
                                headers: [...req.headers],
                                url: new URL(req.url),
                                capturedAt: req.capturedAt,
                                get payload() {
                                  return Promise.resolve(req.requestPayload)
                                },
                              }
                            }

                            return prev
                          })

                          // save the request to the database (with the payload, for the future use)
                          db.createRequest({
                            sID,
                            rID: req.uuid,
                            method: req.method,
                            clientAddress: req.clientAddress,
                            url: req.url.toString(),
                            capturedAt: req.capturedAt,
                            headers: [...req.headers],
                            payload: req.requestPayload,
                          })
                            .then(resolveSlow)
                            .catch(rejectSlow)
                        })
                        .catch(rejectSlow)
                    })
                )
              }
            } else {
              // if the request does not exist in the database, resolve the first (fast) promise with the second one (slow)
              resolveFast(
                // try to get the request from the server
                () =>
                  new Promise<void>((resolveSlow, rejectSlow) => {
                    api
                      .getSessionRequest(sID, rID)
                      .then((req) => {
                        // set the current request with the data from the server (update the state)
                        setRequest({
                          rID: req.uuid,
                          clientAddress: req.clientAddress,
                          method: req.method,
                          headers: [...req.headers],
                          url: req.url,
                          capturedAt: req.capturedAt,
                          get payload() {
                            return Promise.resolve(req.requestPayload)
                          },
                        })

                        // save the request to the database (with the payload, for the future use)
                        db.createRequest({
                          sID,
                          rID: req.uuid,
                          method: req.method,
                          clientAddress: req.clientAddress,
                          url: req.url.toString(),
                          capturedAt: req.capturedAt,
                          headers: [...req.headers],
                          payload: req.requestPayload,
                        })
                          .then(resolveSlow)
                          .catch(rejectSlow)
                      })
                      .catch(rejectSlow)
                  })
              )
            }
          })
          .catch(rejectFast)
      })
    },
    [api, db]
  )

  /**
   * Switch to a request with the given session and request ID.
   *
   * NOTE: The first promise resolves when the request is loaded from the database (FAST), and the second one
   * resolves when the request is loaded from the server (SLOW).
   */
  const switchToRequest = useCallback(
    async (sID: string, rID: string | null): Promise<() => Promise<void>> => {
      if (!rID) {
        setRequest(null)
        return async () => Promise.resolve()
      }

      // TODO: remove request from the database if API returns 404

      // try to get the request from the database (fast)
      const req = await db.getRequest(rID)

      if (req) {
        // If the request exists in the database
        setRequest({
          rID: req.rID,
          clientAddress: req.clientAddress,
          method: req.method,
          headers: [...req.headers],
          url: new URL(req.url),
          capturedAt: req.capturedAt,
          get payload() {
            return Promise.resolve(req.payload)
          },
        })

        // If the payload is already present
        if (req.payload !== null) {
          return async () => Promise.resolve()
        }

        // If the payload is not loaded, get it from the server (slow)
        return async () => {
          const serverReq = await api.getSessionRequest(sID, rID)
          setRequest((prev) =>
            prev
              ? {
                  rID: rID,
                  clientAddress: serverReq.clientAddress,
                  method: serverReq.method,
                  headers: [...serverReq.headers],
                  url: new URL(serverReq.url),
                  capturedAt: serverReq.capturedAt,
                  get payload() {
                    return Promise.resolve(serverReq.requestPayload)
                  },
                }
              : prev
          )
          await db.createRequest({
            sID,
            rID: serverReq.uuid,
            method: serverReq.method,
            clientAddress: serverReq.clientAddress,
            url: serverReq.url.toString(),
            capturedAt: serverReq.capturedAt,
            headers: [...serverReq.headers],
            payload: serverReq.requestPayload,
          })
        }
      } else {
        // If the request is not in the database, load it from the server (slow)
        return async () => {
          const serverReq = await api.getSessionRequest(sID, rID)
          setRequest({
            rID: serverReq.uuid,
            clientAddress: serverReq.clientAddress,
            method: serverReq.method,
            headers: [...serverReq.headers],
            url: serverReq.url,
            capturedAt: serverReq.capturedAt,
            get payload() {
              return Promise.resolve(serverReq.requestPayload)
            },
          })
          await db.createRequest({
            sID,
            rID: serverReq.uuid,
            method: serverReq.method,
            clientAddress: serverReq.clientAddress,
            url: serverReq.url.toString(),
            capturedAt: serverReq.capturedAt,
            headers: [...serverReq.headers],
            payload: serverReq.requestPayload,
          })
        }
      }
    },
    [api, db]
  )

  /**
   * Remove a request with the given session and request ID.
   *
   * NOTE: The first promise resolves when the request is removed from the database (FAST), and the second one
   * resolves when the request is removed from the server (SLOW).
   */
  const removeRequest = useCallback(
    (sID: string, rID: string): Promise<() => Promise<void>> => {
      return new Promise<() => Promise<void>>((resolveFast, rejectFast) => {
        // remove the request from the database (fast)
        db.deleteRequest(rID)
          .then(() => {
            // update the requests list (update the state)
            setRequests((prev) => prev.filter((r) => r.rID !== rID).sort(requestsSorter))

            // remove from the server (slow)
            resolveFast(
              () =>
                new Promise<void>((resolveSlow, rejectSlow) => {
                  api
                    .deleteSessionRequest(sID, rID)
                    .then((ok) => {
                      if (ok) {
                        resolveSlow()
                      } else {
                        rejectSlow(new Error('Failed to delete the request for the session on the server'))
                      }
                    })
                    .catch(rejectSlow)
                })
            )
          })
          .catch(rejectFast)
      })
    },
    [api, db]
  )

  /**
   * Remove all requests for the session with the given ID.
   *
   * NOTE: The first promise resolves when the requests are removed from the database (FAST), and the second one
   * resolves when the requests are removed from the server (SLOW).
   */
  const removeAllRequests = useCallback(
    (sID: string): Promise<() => Promise<void>> => {
      return new Promise<() => Promise<void>>((resolveFast, rejectFast) => {
        // remove all requests from the database
        db.deleteAllRequests(sID)
          .then(() => {
            // clear the requests list (update the state)
            setRequests([])

            resolveFast(
              () =>
                new Promise<void>((resolveSlow, rejectSlow) => {
                  // clear the requests on the server
                  api
                    .deleteAllSessionRequests(sID)
                    .then((ok) => {
                      if (ok) {
                        resolveSlow()
                      } else {
                        rejectSlow(new Error('Failed to delete all requests for the session on the server'))
                      }
                    })
                    .catch(rejectSlow)
                })
            )
          })
          .catch(rejectFast)
      })
    },
    [api, db]
  )

  /** Limit the number of requests by removing the oldest ones, if the count exceeds the limit */
  const setRequestsCount = useCallback((limit: number) => {
    setRequests((prev) => prev.slice(0, limit))
  }, [])

  // on provider mount // TODO: make a separate function and call it somewhere close to app initialization
  useEffect(() => {
    // load all session IDs from the database
    db.getSessionIDs()
      .then((dbSessionIDs) => {
        // set the initial list of session IDs (fast)
        setAllSessionIDs(dbSessionIDs)

        if (dbSessionIDs.length) {
          // if we have any session IDs, check the sessions existence on the server to invalidate the ones that do not
          api
            .checkSessionExists(...dbSessionIDs)
            .then((checkResult) => {
              // filter out the IDs that do not exist on the server
              const toRemove = dbSessionIDs.filter((id) => !checkResult[id])

              // if we have any IDs to remove
              if (toRemove.length) {
                // cleanup the database
                db.deleteSession(...toRemove)
                  .then(() => {
                    // update the list of session IDs (slow)
                    setAllSessionIDs((prev) => prev.filter((id) => !toRemove.includes(id)))
                  })
                  .catch(errHandler)
              }
            })
            .catch(errHandler)
        }
      })
      .catch(errHandler)
  }, [api, db, errHandler])

  // watch for the session changes and update the webhook URL
  useEffect(() => {
    if (session) {
      setWebHookUrl(Object.freeze(new URL(`${window.location.origin}/${session.sID}`)))
    }
  }, [session])

  return (
    <dataContext.Provider
      value={{
        lastUsedSID,
        newSession,
        switchToSession,
        session,
        allSessionIDs,
        destroySession,
        request,
        requests,
        switchToRequest,
        removeRequest,
        removeAllRequests,
        webHookUrl,
        setRequestsCount,
      }}
    >
      {children}
    </dataContext.Provider>
  )
}

export function useData(): DataContext {
  const ctx = useContext(dataContext)

  if (!ctx) {
    throw new Error('useData must be used within a DataProvider')
  }

  return ctx
}

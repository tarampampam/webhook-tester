import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { humanId } from 'human-id'
import { type Client, RequestEventAction } from '~/api'
import { Database } from '~/db'
import { UsedStorageKeys, useStorage } from '~/shared'

export type Session = {
  sID: string
  humanReadableName: string
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
  payload: Uint8Array | null
  capturedAt: Date
}

type DataContext = {
  /** The last used session ID (updates every time a session is switched) */
  lastUsedSID: string | null

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
  }): Promise<Session>

  /** Switch to a session with the given ID. */
  switchToSession(sID: string): Promise<void>

  /** Current active session */
  session: Readonly<Session> | null

  /** The list of all session IDs, available to the user */
  allSessionIDs: ReadonlyArray<string>

  /** Destroy a session with the given ID */
  destroySession(sID: string): Promise<void>

  /** Current active request */
  request: Readonly<Request> | null

  /** The list of requests for the current session, ordered by the captured time (from newest to oldest) */
  requests: ReadonlyArray<Omit<Request, 'payload'>> // omit the payload to reduce the memory usage

  /** Switch to a request with the given ID for the current session */
  switchToRequest(sID: string, rID: string | null): Promise<void>

  /** Remove a request with the given ID for the current session */
  removeRequest(sID: string, rID: string): Promise<void>

  /** Remove all requests for the session with the given ID */
  removeAllRequests(sID: string): Promise<void>

  /** The URL for the webhook (if session is active) */
  webHookUrl: Readonly<URL> | null

  /** The loading state of the session */
  sessionLoading: boolean

  /** The loading state of the request */
  requestLoading: boolean

  /** The loading state of the session requests */
  requestsLoading: boolean
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
  webHookUrl: null,
  sessionLoading: false,
  requestLoading: false,
  requestsLoading: false,
})

// TODO: use notifications for error handling? not sure, since this is a "background" logic
/** Error handler for non-critical errors */
const errHandler = (err: Error | unknown) => console.error(err)

/** Sort requests by the captured time (from newest to oldest) */
const requestsSorter = <T extends { capturedAt: Date }>(a: T, b: T) => b.capturedAt.getTime() - a.capturedAt.getTime()

/**
 * DataProvider is a context provider that manages application data.
 *
 * Think of it as the **core** of the business logic, handling all data and key methods related to sessions and requests.
 */
export const DataProvider = ({ api, db, children }: { api: Client; db: Database; children: React.JSX.Element }) => {
  const [lastUsedSID, setLastUsedSID] = useStorage<string | null>(null, UsedStorageKeys.SessionsLastUsed, 'local')
  const [session, setSession] = useState<Readonly<Session> | null>(null)
  const [allSessionIDs, setAllSessionIDs] = useState<ReadonlyArray<string>>([])
  const [request, setRequest] = useState<Readonly<Request> | null>(null)
  const [requests, setRequests] = useState<ReadonlyArray<Omit<Request, 'payload'>>>([])
  const [webHookUrl, setWebHookUrl] = useState<URL | null>(null)
  const [sessionLoading, setSessionLoading] = useState<boolean>(false)
  const [requestLoading, setRequestLoading] = useState<boolean>(false)
  const [requestsLoading, setRequestsLoading] = useState<boolean>(false)

  // the subscription closer function (if not null, it means the subscription is active)
  const closeSubRef = useRef<(() => void) | null>(null)

  /** Subscribe to the session requests on the server */
  const subscribeToRequestEvents = useCallback(
    (sID: string) => {
      // unsubscribe from the previous session requests
      if (closeSubRef.current) {
        closeSubRef.current()
      }

      closeSubRef.current = null

      return new Promise<void>((resolve, reject) => {
        // subscribe to the session requests on the server
        api
          .subscribeToSessionRequests(sID, {
            onUpdate: (requestEvent): void => {
              switch (requestEvent.action) {
                // a new request was captured
                case RequestEventAction.create: {
                  const req = requestEvent.request

                  if (req) {
                    // append the new request in front of the list
                    setRequests((prev) => [
                      {
                        rID: req.uuid,
                        clientAddress: req.clientAddress,
                        method: req.method,
                        headers: [...req.headers],
                        url: req.url,
                        capturedAt: req.capturedAt,
                      },
                      ...prev,
                    ])

                    // TODO: add limit for the number of requests per session
                    // TODO: show notifications for new requests

                    // save the request to the database
                    db.createRequest({
                      sID: sID,
                      rID: req.uuid,
                      method: req.method,
                      clientAddress: req.clientAddress,
                      url: new URL(req.url),
                      capturedAt: req.capturedAt,
                      headers: [...req.headers],
                    }).catch(errHandler)
                  }

                  break
                }

                // a request was deleted
                case RequestEventAction.delete: {
                  const req = requestEvent.request

                  if (req) {
                    // remove the request from the list
                    setRequests((prev) => prev.filter((r) => r.rID !== req.uuid))

                    // remove the request from the database
                    db.deleteRequest(req.uuid).catch(errHandler)
                  }

                  break
                }

                // all requests were cleared
                case RequestEventAction.clear: {
                  // clear the requests list
                  setRequests([])

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
    [api, db]
  )

  // TODO: remove all useCallbacks - they are **probably** not needed

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
    }): Promise<Session> => {
      return new Promise((resolve, reject) => {
        // save the session to the server
        api
          .newSession({ statusCode, headers, delay, responseBody })
          .then((opts) => {
            const humanReadableName = humanId()

            // save the session to the database
            db.createSession({
              sID: opts.uuid,
              humanReadableName,
              responseCode: statusCode,
              responseDelay: delay,
              responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
              responseBody,
              createdAt: opts.createdAt,
            })
              .then(() => {
                // add the session ID to the list of all session IDs
                setAllSessionIDs((prev) => [...prev, opts.uuid])

                resolve({
                  sID: opts.uuid,
                  humanReadableName,
                  responseCode: statusCode,
                  responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
                  responseDelay: delay,
                  responseBody,
                })
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  /** Load the requests for the session with the given ID */
  const loadRequests = useCallback(
    (sID: string) => {
      return new Promise<void>((resolve, reject) => {
        // load requests for the session from the database (fast)
        db.getSessionRequests(sID).then((reqs) => {
          // update the requests list (first, to show the cached data)
          setRequests(
            reqs
              .map((r) => ({
                rID: r.rID,
                clientAddress: r.clientAddress,
                method: r.method,
                headers: [...r.headers],
                url: r.url,
                capturedAt: r.capturedAt,
              }))
              .sort(requestsSorter)
          )

          setRequestsLoading(true)

          // load requests from the server (slow)
          api
            .getSessionRequests(sID)
            .then((reqs) => {
              // update the requests list (second, to show the fresh data)
              setRequests(
                reqs
                  .map((r) => ({
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
                ...reqs
                  .map((r) => ({
                    sID: sID,
                    rID: r.uuid,
                    method: r.method,
                    clientAddress: r.clientAddress,
                    url: new URL(r.url),
                    capturedAt: r.capturedAt,
                    headers: [...r.headers],
                  }))
                  .sort(requestsSorter)
              ).catch(errHandler)

              resolve()
            })
            .catch(reject)
            .finally(() => setRequestsLoading(false))
        })
      })
    },
    [db, api]
  )

  /** Switch to a session with the given ID. It returns `true` if the session was switched successfully. */
  const switchToSession = useCallback(
    (sID: string) => {
      return new Promise<void>((resolve, reject) => {
        // first, try to find out if the session exists in the database
        db.getSession(sID)
          .then((dbSession) => {
            // if the session exists in the database
            if (dbSession) {
              // set the session as the current session // FIXME: infinite rendering loop occurs here
              setSession({
                sID: dbSession.sID,
                humanReadableName: dbSession.humanReadableName,
                responseCode: dbSession.responseCode,
                responseDelay: dbSession.responseDelay,
                responseHeaders: dbSession.responseHeaders,
                responseBody: dbSession.responseBody,
              })

              // update the last used session ID
              setLastUsedSID(dbSession.sID)

              // load the requests for the session // FIXME: infinite rendering loop occurs here
              loadRequests(dbSession.sID)
                .then(() => subscribeToRequestEvents(dbSession.sID))
                .then(resolve)
                .catch((err) => {
                  setLastUsedSID(null)
                  reject(err)
                })
            } else {
              // otherwise, try to get it from the server
              setSessionLoading(true)

              api
                .getSession(sID)
                .then((apiSession) => {
                  const humanReadableName = humanId()

                  // save the session to the database
                  db.createSession({
                    sID: apiSession.uuid,
                    humanReadableName,
                    responseCode: apiSession.response.statusCode,
                    responseDelay: apiSession.response.delay,
                    responseHeaders: [...apiSession.response.headers],
                    responseBody: apiSession.response.body,
                    createdAt: apiSession.createdAt,
                  })
                    .then(() => {
                      // add the session ID to the list of all session IDs
                      setAllSessionIDs((prev) => [...prev, apiSession.uuid])

                      // set the session as the current session
                      setSession({
                        sID: apiSession.uuid,
                        humanReadableName,
                        responseCode: apiSession.response.statusCode,
                        responseDelay: apiSession.response.delay,
                        responseHeaders: [...apiSession.response.headers],
                        responseBody: apiSession.response.body,
                      })

                      // update the last used session ID
                      setLastUsedSID(apiSession.uuid)

                      // load the requests for the session
                      loadRequests(apiSession.uuid)
                        .then(() => subscribeToRequestEvents(apiSession.uuid))
                        .then(resolve)
                        .catch((err) => {
                          setLastUsedSID(null)
                          reject(err)
                        })
                    })
                    .catch(reject)
                })
                .catch((err) => {
                  setLastUsedSID(null)
                  reject(err)
                })
                .finally(() => setSessionLoading(false))
            }
          })
          .catch((err) => {
            setLastUsedSID(null)
            reject(err)
          })
      })
    },
    [api, db, loadRequests, subscribeToRequestEvents, setLastUsedSID]
  )

  /** Destroy a session with the given ID */
  const destroySession = useCallback(
    (sID: string): Promise<void> => {
      return new Promise((resolve, reject) => {
        // remove the session from the database first
        db.deleteSession(sID)
          .then(() => {
            // remove the session from the list of all session IDs
            setAllSessionIDs((prev) => prev.filter((id) => id !== sID))

            // remove from the server too
            api
              .deleteSession(sID)
              .then((ok) => {
                if (ok) {
                  resolve()
                } else {
                  reject(new Error('Failed to delete the session on the server'))
                }
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  /** Switch to a request with the given ID for the current session */
  const switchToRequest = useCallback(
    (sID: string, rID: string | null): Promise<void> => {
      if (!rID) {
        setRequest(null)

        return Promise.resolve()
      }

      return new Promise<void>((resolve, reject) => {
        // get the request from the database (fast)
        db.getRequest(rID)
          .then((req) => {
            if (req) {
              // set the current request with the data from the database, except the payload
              setRequest({
                rID: req.rID,
                clientAddress: req.clientAddress,
                method: req.method,
                headers: [...req.headers],
                url: req.url,
                payload: null, // database does not store the payload
                capturedAt: req.capturedAt,
              })

              setRequestLoading(true)

              // get the request payload from the server (slow)
              api
                .getSessionRequest(sID, rID)
                .then((req) => {
                  setRequest((prev) => {
                    if (prev) {
                      return { ...prev, payload: req.requestPayload }
                    }

                    return prev
                  })
                })
                .then(resolve)
                .catch(reject)
                .finally(() => setRequestLoading(false))
            } else {
              setRequestLoading(true)

              // if the request does not exist in the database, try to get it from the server
              api
                .getSessionRequest(sID, rID)
                .then((req) => {
                  // set the current request with the data from the server
                  setRequest({
                    rID: req.uuid,
                    clientAddress: req.clientAddress,
                    method: req.method,
                    headers: [...req.headers],
                    url: req.url,
                    payload: req.requestPayload,
                    capturedAt: req.capturedAt,
                  })

                  // save the request to the database
                  db.createRequest({
                    sID,
                    rID: req.uuid,
                    method: req.method,
                    clientAddress: req.clientAddress,
                    url: new URL(req.url),
                    capturedAt: req.capturedAt,
                    headers: [...req.headers],
                  })
                    .then(resolve)
                    .catch(reject)
                })
                .catch(reject)
                .finally(() => setRequestLoading(false))
            }
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  /** Remove a request with the given ID for the current session */
  const removeRequest = useCallback(
    (sID: string, rID: string): Promise<void> => {
      return new Promise<void>((resolve, reject) => {
        // remove the request from the database
        db.deleteRequest(rID)
          .then(() => {
            // update the requests list
            setRequests((prev) => prev.filter((r) => r.rID !== rID).sort(requestsSorter))

            // remove from the server, if session is active
            api
              .deleteSessionRequest(sID, rID)
              .then((ok) => {
                if (ok) {
                  resolve()
                } else {
                  reject(new Error('Failed to delete the request for the session on the server'))
                }
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  // on provider mount
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
                    // update the list of session IDs
                    setAllSessionIDs((prev) => prev.filter((id) => !toRemove.includes(id)))
                  })
                  .catch(errHandler)
              }
            })
            .catch(errHandler)
        }
      })
      .catch(errHandler)
  }, [api, db])

  /** Remove all requests for the session with the given ID */
  const removeAllRequests = useCallback(
    (sID: string): Promise<void> => {
      return new Promise<void>((resolve, reject) => {
        // remove all requests from the database
        db.deleteAllRequests(sID)
          .then(() => {
            // clear the requests list
            setRequests([])

            // clear the requests on the server
            api
              .deleteAllSessionRequests(sID)
              .then((ok) => {
                if (ok) {
                  resolve()
                } else {
                  reject(new Error('Failed to delete all requests for the session on the server'))
                }
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

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
        sessionLoading,
        requestLoading,
        requestsLoading,
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

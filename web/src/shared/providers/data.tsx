import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { APIErrorNotFound, type Client, RequestEventAction } from '~/api'
import { Database } from '~/db'
import { UsedStorageKeys, useSettings, useStorage } from '~/shared'

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

export type SessionEvents = {
  onNewRequest: (r: Omit<Request, 'payload'>) => void // server does not send the payload
  onRequestDelete: (r: Omit<Request, 'payload'>) => void // server does not send the payload
  onRequestsClear: () => void
  onError: (err: Error | unknown) => void
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
  removeRequest(sID: string, rID: string, andFromServer?: boolean): Promise<() => Promise<void>>

  /**
   * Remove all requests for the session with the given ID.
   *
   * NOTE: The first promise resolves when the requests are removed from the database (FAST), and the second one
   * resolves when the requests are removed from the server (SLOW).
   */
  removeAllRequests(sID: string, andFromServer?: boolean): Promise<() => Promise<void>>

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
  const { publicUrlRoot } = useSettings()
  const [lastUsedSID, setLastUsedSID] = useStorage<string | null>(null, UsedStorageKeys.SessionsLastUsed, 'local')
  const [session, setSession] = useState<Readonly<Session> | null>(null)
  const [allSessionIDs, setAllSessionIDs] = useState<ReadonlyArray<string>>([])
  const [request, setRequest] = useState<Readonly<Request> | null>(null)
  const [requests, setRequests] = useState<ReadonlyArray<Request>>([])
  const [webHookUrl, setWebHookUrl] = useState<URL | null>(null)

  // the subscription closer function (if not null, it means the subscription is active)
  const closeSubRef = useRef<(() => void) | null>(null)

  /** Unsubscribe from the session requests on the server */
  const unsubscribe = (): void => {
    if (closeSubRef.current) {
      closeSubRef.current()
    }

    closeSubRef.current = null
  }

  /** Subscribe to the session requests on the server */
  const subscribeToRequestEvents = useCallback(
    async (sID: string, listeners?: Partial<SessionEvents>) => {
      // terminate the previous subscription, if any
      unsubscribe()

      // subscribe to the session requests on the server
      closeSubRef.current = await api.subscribeToSessionRequests(sID, {
        onUpdate: async (requestEvent): Promise<void> => {
          try {
            switch (requestEvent.action) {
              // a new request was captured
              case RequestEventAction.create: {
                const req = requestEvent.request

                if (req) {
                  // save the request to the database (without payload)
                  await db.putRequest({
                    sID,
                    rID: req.uuid,
                    method: req.method,
                    clientAddress: req.clientAddress,
                    url: req.url.toString(),
                    payload: null, // server does not send the payload
                    capturedAt: req.capturedAt,
                    headers: [...req.headers],
                  })

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
                  listeners?.onRequestDelete?.(
                    Object.freeze({
                      rID: req.uuid,
                      clientAddress: req.clientAddress,
                      method: req.method,
                      headers: [...req.headers],
                      url: req.url,
                      capturedAt: req.capturedAt,
                    })
                  )

                  // remove the request from the database
                  await db.deleteRequest(req.uuid)
                }

                break
              }

              // all requests were cleared
              case RequestEventAction.clear: {
                // clear the requests list
                setRequests(Object.freeze([]))

                // invoke the listener callback
                listeners?.onRequestsClear?.()

                // clear the requests from the database
                await db.deleteAllRequests(sID)

                break
              }
            }
          } catch (err) {
            if (listeners?.onError) {
              listeners.onError(err)
            } else {
              throw err
            }
          }
        },
        onError: (err) => {
          if (listeners?.onError) {
            listeners.onError(err)
          } else {
            throw err
          }
        },
      })
    },
    [api, db]
  )

  /** Create a new session */
  const newSession = useCallback(
    async ({
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
      // save the session to the server
      const opts = await api.newSession({ statusCode, headers, delay, responseBody })

      // add the session ID to the list of all session IDs (update the state)
      setAllSessionIDs((prev) => [...prev, opts.uuid])

      // save the session to the database
      await db.putSession({
        sID: opts.uuid,
        responseCode: statusCode,
        responseDelay: delay,
        responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
        responseBody,
        createdAt: opts.createdAt,
      })

      return Object.freeze({
        sID: opts.uuid,
        responseCode: statusCode,
        responseHeaders: Object.entries(headers).map(([name, value]) => ({ name, value })),
        responseDelay: delay,
        responseBody,
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
    async (sID: string): Promise<() => Promise<void>> => {
      // load requests for the session from the database (fast)
      const dbList = await db.getSessionRequests(sID)

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

      // return a function that loads requests from the server (slow)
      return async () => {
        const reqs = await api.getSessionRequests(sID)

        // update the requests list (second state update, to show the fresh data)
        setRequests(
          reqs
            .map((r) =>
              Object.freeze({
                ...payloadGetter(db, r.uuid),
                rID: r.uuid,
                clientAddress: r.clientAddress,
                method: r.method,
                headers: [...r.headers],
                url: r.url,
                capturedAt: r.capturedAt,
              })
            )
            .sort(requestsSorter)
        )

        // update the requests in the database (for future use)
        await db.putRequest(
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

        // find requests that are not present in the server response but are in the database
        const toRemove = dbList.filter((r) => !reqs.find((req) => req.uuid === r.rID)).map((r) => r.rID)

        if (toRemove.length) {
          await db.deleteRequest(...toRemove)
        }
      }
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
    async (sID: string, listeners?: Partial<SessionEvents>) => {
      try {
        // try to find out if the session exists in the database
        const dbSession = await db.getSession(sID)

        if (dbSession) {
          // if the session exists in the database
          setSession(
            Object.freeze({
              sID: dbSession.sID,
              responseCode: dbSession.responseCode,
              responseDelay: dbSession.responseDelay,
              responseHeaders: dbSession.responseHeaders,
              responseBody: dbSession.responseBody,
            })
          )

          setLastUsedSID(dbSession.sID)

          // load requests for the session
          const requestsSlow = await loadRequests(dbSession.sID)

          // return a function that resolves the second (slow) promise
          return async () => {
            try {
              await requestsSlow()
              await subscribeToRequestEvents(dbSession.sID, listeners)
            } catch (err) {
              unsubscribe() // unsubscribe from the session requests if something went wrong
              setLastUsedSID(null) // unset the last used session ID
              setSession(null) // clear the session state

              throw err // reject the second (slow) promise
            }
          }
        } else {
          // otherwise, load the session from the server
          return async () => {
            try {
              const apiSession = await api.getSession(sID)

              // save the session to the database
              await db.putSession({
                sID: apiSession.uuid,
                responseCode: apiSession.response.statusCode,
                responseDelay: apiSession.response.delay,
                responseHeaders: [...apiSession.response.headers],
                responseBody: apiSession.response.body,
                createdAt: apiSession.createdAt,
              })

              setAllSessionIDs((prev) => [...prev, apiSession.uuid])

              setSession(
                Object.freeze({
                  sID: apiSession.uuid,
                  responseCode: apiSession.response.statusCode,
                  responseDelay: apiSession.response.delay,
                  responseHeaders: [...apiSession.response.headers],
                  responseBody: apiSession.response.body,
                })
              )

              setLastUsedSID(apiSession.uuid)

              // load requests for the session
              const requestsSlow = await loadRequests(apiSession.uuid)

              // load requests from the server and subscribe to the session requests
              await requestsSlow()
              await subscribeToRequestEvents(apiSession.uuid, listeners)
            } catch (err) {
              unsubscribe() // unsubscribe from the session requests if something went wrong
              setLastUsedSID(null) // unset the last used session ID
              setSession(null) // clear the session state

              throw err // reject the second (slow) promise
            }
          }
        }
      } catch (err) {
        setLastUsedSID(null) // unset the last used session ID
        setSession(null) // clear the session state

        throw err // reject the first (fast) promise
      }
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
    async (sID: string): Promise<() => Promise<void>> => {
      // remove the session from the database first (fast)
      await db.deleteSession(sID)

      // update the session list state
      setAllSessionIDs((prev) => prev.filter((id) => id !== sID))

      // return a function to remove the session from the server (slow)
      return async () => {
        const ok = await api.deleteSession(sID)

        if (!ok) {
          throw new Error('Failed to delete the session on the server')
        }
      }
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
        // if the request exists in the database
        setRequest(
          Object.freeze({
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
        )

        // if the payload is already present
        if (req.payload !== null) {
          return async () => Promise.resolve()
        }

        // if the payload is not loaded, get it from the server (slow)
        return async () => {
          try {
            const serverReq = await api.getSessionRequest(sID, rID)

            // update the state with the actual data from the server including the payload
            setRequest((prev) =>
              prev
                ? Object.freeze({
                    rID: rID,
                    clientAddress: serverReq.clientAddress,
                    method: serverReq.method,
                    headers: [...serverReq.headers],
                    url: new URL(serverReq.url),
                    capturedAt: serverReq.capturedAt,
                    get payload() {
                      return Promise.resolve(serverReq.requestPayload)
                    },
                  })
                : prev
            )

            await db.putRequest({
              sID,
              rID: serverReq.uuid,
              method: serverReq.method,
              clientAddress: serverReq.clientAddress,
              url: serverReq.url.toString(),
              capturedAt: serverReq.capturedAt,
              headers: [...serverReq.headers],
              payload: serverReq.requestPayload,
            })
          } catch (err) {
            // if the request is not found on the server
            if (err instanceof APIErrorNotFound) {
              // remove it from the database
              await db.deleteRequest(rID)

              // update the requests list (update the state)
              setRequests((prev) => prev.filter((r) => r.rID !== rID).sort(requestsSorter))

              // clear the request state
              setRequest(null)
            } else {
              throw err
            }
          }
        }
      } else {
        // if the request is not in the database, load it from the server (slow)
        return async () => {
          const serverReq = await api.getSessionRequest(sID, rID)

          setRequest(
            Object.freeze({
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
          )

          await db.putRequest({
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
    async (sID: string, rID: string, andFromServer: boolean = true): Promise<() => Promise<void>> => {
      // remove the request from the database (fast)
      await db.deleteRequest(rID)

      // update the requests list (update the state)
      setRequests((prev) => prev.filter((r) => r.rID !== rID).sort(requestsSorter))

      // skip the slow operation if we don't need to remove the request from the server
      if (!andFromServer) {
        return async () => Promise.resolve()
      }

      // return a function to remove from the server (slow)
      return async () => {
        const ok = await api.deleteSessionRequest(sID, rID)

        if (!ok) {
          throw new Error('Failed to delete the request for the session on the server')
        }
      }
    },
    [api, db]
  )

  /**
   * Remove all requests for the session with the given ID.
   *
   * NOTE: The first operation resolves when the requests are removed from the database (FAST), and the second one
   * resolves when the requests are removed from the server (SLOW).
   */
  const removeAllRequests = useCallback(
    async (sID: string, andFromServer: boolean = true): Promise<() => Promise<void>> => {
      // remove all requests from the database
      await db.deleteAllRequests(sID)

      // clear the requests list (update the state)
      setRequests(Object.freeze([]))

      // skip the slow operation if we don't need to remove the request from the server
      if (!andFromServer) {
        return async () => Promise.resolve()
      }

      // return the function that removes requests from the server
      return async (): Promise<void> => {
        const ok = await api.deleteAllSessionRequests(sID)

        if (!ok) {
          throw new Error('Failed to delete all requests for the session on the server')
        }
      }
    },
    [api, db]
  )

  /** Limit the number of requests by removing the oldest ones, if the count exceeds the limit */
  const setRequestsCount = useCallback((limit: number) => {
    setRequests((prev) => prev.slice(0, limit))
  }, [])

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
                // if all sessions from the database are to be removed (we have no any sessions left)
                if (dbSessionIDs.filter((id) => !toRemove.includes(id)).length === 0) {
                  // clear the state
                  setSession(null)
                  setRequest(null)
                  setLastUsedSID(null)
                }

                // cleanup the database
                db.deleteSession(...toRemove)
                  // update the list of session IDs (slow)
                  .then(() => setAllSessionIDs((prev) => prev.filter((id) => !toRemove.includes(id))))
                  .catch(errHandler)
              }
            })
            .catch(errHandler)
        }
      })
      .catch(errHandler)
  }, [api, db, errHandler, setLastUsedSID])

  // watch for the session changes and update the webhook URL
  useEffect(() => {
    if (session) {
      const baseUrl = publicUrlRoot ? publicUrlRoot.origin : window.location.origin
      setWebHookUrl(Object.freeze(new URL(`${baseUrl}/${session.sID}`)))
    }
  }, [session, publicUrlRoot])

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

import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { humanId } from 'human-id'
import { type Client, RequestEventAction } from '~/api'
import { Database } from '~/db'
import { UsedStorageKeys, useStorage } from '~/shared'

export type Session = {
  sID: string
  humanReadableName: string
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

  /** Switch to a session with the given ID. It returns `true` if the session was switched successfully. */
  switchToSession(sID: string): Promise<boolean>

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
  switchToRequest(rID: string): Promise<boolean>

  /** Remove a request with the given ID for the current session */
  removeRequest(rID: string): Promise<void>
}

const notInitialized = () => {
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
})

// TODO: use notifications for error handling?
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

  // the subscription closer function (if not null, it means the subscription is active)
  const closeSubRef = useRef<(() => void) | null>(null)

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
            db.createSession({ sID: opts.uuid, humanReadableName, createdAt: opts.createdAt })
              .then(() => {
                // add the session ID to the list of all session IDs
                setAllSessionIDs((prev) => [...prev, opts.uuid])
                // empty the requests list
                setRequests([])

                resolve({ sID: opts.uuid, humanReadableName })
              })
              .catch(reject)
          })
          .catch(reject)
      })
    },
    [api, db]
  )

  /** Switch to a session with the given ID. It returns `true` if the session was switched successfully. */
  const switchToSession = useCallback(
    (sID: string) => {
      return new Promise<boolean>((resolve, reject) => {
        // subscribe to the session requests
        if (closeSubRef.current) {
          closeSubRef.current()
        }

        // get the session from the database
        db.getSession(sID)
          .then((opts) => {
            if (opts) {
              // set the session as the current session
              setSession({ sID: opts.sID, humanReadableName: opts.humanReadableName })
              // update the last used session ID
              setLastUsedSID(opts.sID)

              // load requests for the session from the database (fast)
              db.getSessionRequests(opts.sID)
                .then((reqs) => {
                  setRequests(
                    reqs
                      .map((r) => ({
                        rID: r.rID,
                        clientAddress: r.clientAddress,
                        method: r.method,
                        headers: r.headers.map((h) => ({ name: h.name, value: h.value })),
                        url: r.url,
                        capturedAt: r.capturedAt,
                      }))
                      .sort(requestsSorter)
                  )

                  // load requests from the server (slow)
                  api
                    .getSessionRequests(opts.sID)
                    .then((reqs) => {
                      setRequests(
                        reqs
                          .map((r) => ({
                            rID: r.uuid,
                            clientAddress: r.clientAddress,
                            method: r.method,
                            headers: r.headers.map((h) => ({ name: h.name, value: h.value })),
                            url: r.url,
                            capturedAt: r.capturedAt,
                          }))
                          .sort(requestsSorter)
                      )

                      // update the requests in the database (for the future use)
                      db.createRequest(
                        ...reqs
                          .map((r) => ({
                            sID: opts.sID,
                            rID: r.uuid,
                            method: r.method,
                            clientAddress: r.clientAddress,
                            url: new URL(r.url),
                            capturedAt: r.capturedAt,
                            headers: r.headers.map((h) => ({ name: h.name, value: h.value })),
                          }))
                          .sort(requestsSorter)
                      ).catch(errHandler)
                    })
                    .catch(errHandler)

                  // subscribe to the session requests on the server
                  api
                    .subscribeToSessionRequests(opts.sID, {
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
                                  headers: req.headers.map((h) => ({ name: h.name, value: h.value })),
                                  url: req.url,
                                  capturedAt: req.capturedAt,
                                },
                                ...prev,
                              ])

                              // TODO: add limit for the number of requests per session
                              // TODO: show notifications for new requests

                              // save the request to the database
                              db.createRequest({
                                sID: opts.sID,
                                rID: req.uuid,
                                method: req.method,
                                clientAddress: req.clientAddress,
                                url: new URL(req.url),
                                capturedAt: req.capturedAt,
                                headers: req.headers.map((h) => ({ name: h.name, value: h.value })),
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
                            db.deleteAllRequests(opts.sID).catch(errHandler)

                            break
                          }
                        }
                      },
                      onError: (error) => errHandler(error),
                    })
                    .then((closer) => (closeSubRef.current = closer))
                    .catch(errHandler)
                })
                .catch(errHandler)

              return resolve(true)
            }

            return resolve(false)
          })
          .catch(reject)
      })
    },
    [api, db, setLastUsedSID]
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

            // if the session is the current session, unset the current session
            if (!!session && session.sID === sID) {
              setSession(null)
            }

            // remove from the server too
            api.deleteSession(sID).catch(errHandler)

            resolve()
          })
          .catch(reject)
      })
    },
    [api, db, session]
  )

  /** Switch to a request with the given ID for the current session */
  const switchToRequest = useCallback(
    (rID: string): Promise<boolean> => {
      if (!session) {
        return Promise.resolve(false)
      }

      return new Promise((resolve, reject) => {
        // get the request from the database (fast)
        db.getRequest(rID)
          .then((req) => {
            if (req) {
              // set the current request with the data from the database, except the payload
              setRequest({
                rID: req.rID,
                clientAddress: req.clientAddress,
                method: req.method,
                headers: req.headers.map((h) => ({ name: h.name, value: h.value })),
                url: req.url,
                payload: null, // database does not store the payload
                capturedAt: req.capturedAt,
              })

              // get the request payload from the server (slow)
              api
                .getSessionRequest(session.sID, rID)
                .then((req) => {
                  setRequest((prev) => {
                    if (prev) {
                      return { ...prev, payload: req.requestPayload }
                    }

                    return prev
                  })
                })
                .catch(reject)
            } else {
              resolve(false)
            }
          })
          .catch(reject)
      })
    },
    [api, db, session]
  )

  /** Remove a request with the given ID for the current session */
  const removeRequest = useCallback(
    (rID: string): Promise<void> => {
      return new Promise((resolve, reject) => {
        // remove the request from the database
        db.deleteRequest(rID)
          .then(() => {
            // update the requests list
            setRequests((prev) => prev.filter((r) => r.rID !== rID).sort(requestsSorter))

            // remove from the server, if session is active
            if (session) {
              api.deleteSessionRequest(session.sID, rID).catch(errHandler)
            }

            resolve()
          })
          .catch(reject)
      })
    },
    [api, db, session]
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

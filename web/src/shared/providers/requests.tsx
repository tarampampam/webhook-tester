import React, {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react'
import { type Client, RequestEventAction } from '~/api'
import type { Database, RequestInput } from '~/db'
import { anyToError } from '../utils/errors'
import { useAppConfig } from './app-config'

/** A single captured HTTP request, as held in the requests provider's state. */
export type Request = {
  readonly sID: string
  readonly id: string
  readonly clientAddress: string
  readonly method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS' | 'CONNECT' | 'TRACE' | string
  headers: ReadonlyArray<{ name: string; value: string }>
  url: Readonly<URL>
  getPayload: () => Promise<Uint8Array | null>
  capturedAt: Readonly<Date>
}

type Options = {
  readonly signal?: AbortSignal
}

/** Internal shape exposed by RequestsProvider through context and consumed by useRequests. */
interface Context {
  /**
   * All captured requests for the active session, keyed by request UUID. Kept in sync via WebSocket events.
   */
  requests: ReadonlyMap<string, Request>

  /**
   * Activate a session: hydrate state from IndexedDB immediately, then reconcile against the API and open
   * a WebSocket for real-time updates. Tears down any previously active session first.
   */
  setSessionID(sID: string): Promise<void>

  /**
   * Tear down the active session: abort all in-flight work, close the WebSocket, and clear the requests map.
   */
  unsetSessionID(): void

  /**
   * Delete a single request via the API, then remove it from IndexedDB and local state. No-op if the server
   * returns false.
   */
  delRequest(sID: string, rID: string, opts?: Options): Promise<void>

  /**
   * Delete all requests for a session via the API, then remove them from IndexedDB and local state. No-op if
   * the server returns false.
   */
  delAllRequests(sID: string, opts?: Options): Promise<void>
}

const ctx = createContext<Context | null>(null)

type RequestsProviderProps = {
  api: Client
  db: Database
  errHandler?: (err: Error) => void
}

/**
 * Returns a new map with only the `limit` most-recent entries.
 *
 * Returns the same object (reference) if no trim needed.
 */
const applyRequestsLimit = (
  m: ReadonlyMap<string, Request>,
  limit: number | undefined
): ReadonlyMap<string, Request> => {
  if (!limit || limit <= 0 || m.size <= limit) {
    return m
  }

  const sorted = [...m.entries()].sort(([, a], [, b]): number => a.capturedAt.getTime() - b.capturedAt.getTime())

  return new Map(sorted.slice(sorted.length - limit))
}

/**
 * Context provider that manages the requests list.
 *
 * @note errHandler must be stabilized by the caller (e.g. with useCallback) to avoid
 *       unnecessary re-renders and effect re-runs.
 */
export const RequestsProvider = ({
  api,
  db,
  requestsLimit,
  errHandler,
  children,
}: PropsWithChildren<RequestsProviderProps & { requestsLimit: number | undefined }>): React.JSX.Element => {
  const [requests, setRequests] = useState<ReadonlyMap<string, Request>>(new Map())
  const ctrlRef = useRef<AbortController | null>(null)
  const closeRef = useRef<(() => void) | null>(null)
  const limitRef = useRef<number | undefined>(requestsLimit)
  const sIDRef = useRef<string | null>(null)

  const subscribe = useCallback(
    async (sID: string): Promise<void> => {
      closeRef.current?.()
      closeRef.current = null

      // capture signal before the first await - ctrlRef.current may be replaced by the time we resume
      const signal = ctrlRef.current?.signal
      const close = await api.subscribeToSessionRequests(
        sID,
        {
          onUpdate: (event): void => {
            void (async (): Promise<void> => {
              switch (event.action) {
                case RequestEventAction.create: {
                  const r = event.request
                  if (!r) {
                    break // should never happen - created events must have a request
                  }

                  if (signal?.aborted) {
                    return
                  }

                  await db.putRequests(
                    [
                      {
                        sID,
                        rID: r.uuid,
                        method: r.method,
                        clientAddress: r.clientAddress,
                        url: r.url.toString(),
                        capturedAt: r.capturedAt,
                        headers: r.headers,
                        payload: undefined, // payload is not coming through the websocket
                      },
                    ],
                    limitRef.current
                  )

                  if (signal?.aborted) {
                    return
                  }

                  const requestUrl = new URL(r.url)

                  // update the state with the new request (but without payload yet)
                  setRequests((prev) => {
                    if (prev.has(r.uuid)) {
                      return prev // already have this request - no update needed
                    }

                    const next = new Map(prev)
                    next.set(r.uuid, {
                      sID,
                      id: r.uuid,
                      clientAddress: r.clientAddress,
                      method: r.method,
                      headers: r.headers,
                      url: requestUrl,
                      capturedAt: r.capturedAt,
                      getPayload: async (): Promise<null> => null,
                    })

                    return applyRequestsLimit(next, limitRef.current)
                  })

                  // get the captured request payload from the api
                  const full = await api.getSessionRequest(sID, r.uuid, { signal })
                  if (signal?.aborted) {
                    return
                  }

                  // update the db with the payload - this will make it available to the getPayload function in the state
                  await db.putRequestPayload(r.uuid, full.requestPayload)

                  // update the state to notify consumers that the payload is now available
                  setRequests((prev) => {
                    const existing = prev.get(r.uuid)
                    if (!existing) {
                      return prev
                    }

                    const next = new Map(prev)
                    next.set(r.uuid, {
                      ...existing,
                      getPayload: db.newRequestPayloadReader(r.uuid), // deliver payload directly from the db
                    })

                    return next
                  })

                  break
                }
                case RequestEventAction.delete: {
                  const r = event.request
                  if (!r) {
                    break // should never happen - deleted events must have a request
                  }

                  await db.deleteRequest(r.uuid)

                  // update the state to remove the deleted request
                  setRequests((prev) => {
                    if (!prev.has(r.uuid)) {
                      return prev // don't have this request - no update needed
                    }

                    const next = new Map(prev)
                    next.delete(r.uuid)

                    return next
                  })

                  break
                }
                case RequestEventAction.clear: {
                  await db.deleteAllRequests(sID)
                  setRequests(new Map())

                  break
                }
              }
            })().catch((err) => {
              if (err instanceof DOMException && err.name === 'AbortError') {
                return
              }

              if (errHandler) {
                errHandler(anyToError(err))
              }
            })
          },
          onError: (err) => {
            if (err instanceof DOMException && err.name === 'AbortError') {
              return
            }

            if (errHandler) {
              errHandler(anyToError(err))
            }
          },
        },
        { signal }
      )

      if (signal?.aborted) {
        close()

        return
      }

      closeRef.current = close
    },
    [api, db, errHandler]
  )

  const unsetSessionID = useCallback((): void => {
    ctrlRef.current?.abort()
    ctrlRef.current = null
    closeRef.current?.()
    closeRef.current = null
    sIDRef.current = null

    setRequests(new Map())
  }, [])

  const setSessionID = useCallback(
    async (sID: string): Promise<void> => {
      unsetSessionID()

      const ctrl = new AbortController()
      ctrlRef.current = ctrl
      const { signal } = ctrl

      sIDRef.current = sID

      try {
        const dbData = await db.getRequests(sID)
        if (signal.aborted) {
          return
        }

        // set requests from the db (FAST)
        setRequests(() => {
          const m = new Map<string, Request>()
          for (const r of dbData) {
            m.set(r.rID, {
              sID,
              id: r.rID,
              clientAddress: r.clientAddress,
              method: r.method,
              headers: r.headers,
              url: new URL(r.url),
              capturedAt: r.capturedAt,
              getPayload: r.getPayload,
            })
          }

          return applyRequestsLimit(m, limitRef.current)
        })

        // fetch requests from the api (SLOW)
        const apiData = await api.getSessionRequests(sID)
        if (signal.aborted) {
          return
        }

        // remove stale db entries (exist in db but not in api response)
        const apiIds = new Set(apiData.map((r) => r.uuid))
        const staleIds = dbData.filter((r) => !apiIds.has(r.rID)).map((r) => r.rID)
        if (staleIds.length) {
          await db.deleteRequest(...staleIds)
        }

        if (signal.aborted) {
          return
        }

        // sync db with the api data
        if (apiData.length) {
          await db.putRequests(
            apiData.map<RequestInput>((r) => ({
              sID,
              rID: r.uuid,
              method: r.method,
              clientAddress: r.clientAddress,
              url: r.url.toString(),
              capturedAt: r.capturedAt,
              headers: r.headers,
              payload: r.requestPayload,
            })),
            limitRef.current
          )
        }

        if (signal.aborted) {
          return
        }

        const parsedUrls = new Map(apiData.map((r) => [r.uuid, new URL(r.url)]))

        // reconcile state: api is authoritative - add new, remove stale
        setRequests((prev) => {
          const next = new Map<string, Request>()
          for (const r of apiData) {
            const url = parsedUrls.get(r.uuid)
            if (!url) {
              continue
            }

            next.set(
              r.uuid,
              prev.get(r.uuid) ?? {
                sID,
                id: r.uuid,
                clientAddress: r.clientAddress,
                method: r.method,
                headers: r.headers,
                url,
                capturedAt: r.capturedAt,
                getPayload: db.newRequestPayloadReader(r.uuid), // deliver payload directly from the db
              }
            )
          }

          // avoid re-render if nothing changed (same keys, same count)
          if (next.size === prev.size && [...next.keys()].every((k) => prev.has(k))) {
            return prev
          }

          return applyRequestsLimit(next, limitRef.current)
        })

        // subscribe to real-time updates only after initial state is settled
        await subscribe(sID)
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        if (errHandler) {
          errHandler(anyToError(err))
        }
      }
    },
    [unsetSessionID, db, api, subscribe, errHandler]
  )

  const delRequest = useCallback(
    async (sID: string, rID: string, opts?: Options): Promise<void> => {
      if (await api.deleteSessionRequest(sID, rID, { signal: opts?.signal })) {
        await db.deleteRequest(rID)

        setRequests((prev) => {
          if (!prev.has(rID)) {
            return prev // don't have this request - no update needed
          }

          const next = new Map(prev)
          next.delete(rID)

          return next
        })
      }
    },
    [db, api]
  )

  const delAllRequests = useCallback(
    async (sID: string, opts?: Options): Promise<void> => {
      if (await api.deleteAllSessionRequests(sID, { signal: opts?.signal })) {
        await db.deleteAllRequests(sID)

        setRequests((prev) => {
          // ensure current session's requests are linked to the sID before clearing
          if (![...prev.values()].some((r) => r.sID === sID)) {
            return prev // no requests for this session - no update needed
          }

          const next = new Map(prev)
          for (const [rID, r] of prev) {
            if (r.sID === sID) {
              next.delete(rID)
            }
          }

          return next
        })
      }
    },
    [db, api]
  )

  // keep ref in sync with prop and retroactively trim current state whenever the limit changes
  useEffect(() => {
    limitRef.current = requestsLimit

    const sID = sIDRef.current
    if (sID && requestsLimit) {
      db.trimRequests(sID, requestsLimit)
        .then(() => setRequests((prev) => applyRequestsLimit(prev, requestsLimit)))
        .catch((err) => {
          if (errHandler) {
            errHandler(anyToError(err))
          }
        })
    }
  }, [db, errHandler, requestsLimit])

  return (
    <ctx.Provider value={{ requests, setSessionID, unsetSessionID, delRequest, delAllRequests }}>
      {children}
    </ctx.Provider>
  )
}

/**
 * Higher-order component that wraps RequestsProvider and injects the requests limit from the app config.
 */
const WithConfig = ({ children, ...props }: PropsWithChildren<RequestsProviderProps>): React.JSX.Element => {
  const cfg = useAppConfig()

  return (
    <RequestsProvider requestsLimit={cfg.config?.limits.maxRequests} {...props}>
      {children}
    </RequestsProvider>
  )
}

RequestsProvider.WithConfig = WithConfig // attach the HOC as a static property for convenient access

/**
 * Hook for accessing the current session's captured requests and session controls.
 * Throws if called outside a RequestsProvider.
 */
export const useRequests = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useRequests must be used within a RequestsProvider')
  }

  return context
}

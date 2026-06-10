import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { type Client } from '~/api'
import { type Database } from '~/db'
import { anyToError } from '~/shared'

/** Session as seen by the UI - holds response config and a lazy body getter. */
export type Session = {
  id: string
  response: {
    readonly code: number
    headers: ReadonlyArray<{ name: string; value: string }>
    readonly delay: number
    getBody: () => Promise<Uint8Array | null>
  }
}

/** Options for creating a new session. All fields are optional. */
interface NewSessionOptions {
  readonly statusCode?: number
  readonly headers?: Record<string, string>
  readonly delay?: number
  readonly responseBody?: Uint8Array
}

const NEW_SESSION_DEFAULTS: Required<NewSessionOptions> = {
  statusCode: 200,
  headers: {},
  delay: 0,
  responseBody: new Uint8Array(),
}

type Options = {
  readonly signal?: AbortSignal
}

interface Context {
  /**
   * Current snapshot of all tracked sessions, keyed by session UUID. Produces a new Map reference on every change
   * so React re-render comparisons work correctly.
   *
   * Populated optimistically from IndexedDB on mount, then reconciled against the backend.
   */
  sessions: ReadonlyMap<string, Session>

  /**
   * Creates a new session. Registers it on the backend first, then persists to IndexedDB and updates local state.
   * Returns the new session UUID.
   *
   * The caller is responsible for passing an AbortSignal tied to the component's lifetime.
   */
  newSession(userOptions: NewSessionOptions, reqOpts?: Options): Promise<string>

  /**
   * Deletes a session by UUID. Removes it from IndexedDB and local state immediately (optimistic), then fires the
   * backend delete in the background (slow).
   *
   * The UI reflects the removal before the network call completes, so there is no rollback on backend failure.
   *
   * The caller is responsible for passing an AbortSignal tied to the component's lifetime.
   */
  delSession(id: string, opts?: Options): Promise<void>
}

const ctx = createContext<Context | null>(null)

/**
 * Context provider that manages the session list.
 *
 * @note errHandler must be stabilized by the caller (e.g. with useCallback) to avoid
 *       unnecessary re-renders and effect re-runs.
 */
export const SessionsProvider = ({
  api,
  db,
  errHandler,
  children,
}: {
  api: Client
  db: Database
  errHandler?: (err: Error) => void
  children?: React.ReactNode
}): React.JSX.Element => {
  const [sessions, setSessions] = useState<Map<string, Session>>(new Map())

  const newSession = useCallback(
    async (userOptions: NewSessionOptions, reqOpts?: Options): Promise<string> => {
      const opts = { ...NEW_SESSION_DEFAULTS, ...userOptions }
      const meta = await api.newSession(opts, { signal: reqOpts?.signal }) // create session on the backend (SLOW)
      const hdrs = Object.entries(opts.headers).map(([name, value]) => ({ name, value }))

      // create session in database (FAST)
      await db.putSession({
        id: meta.uuid,
        response: {
          code: opts.statusCode,
          headers: hdrs,
          delay: opts.delay,
          body: opts.responseBody,
        },
        createdAt: meta.createdAt,
      })

      // update state with the new session
      setSessions((prev) => {
        const next = new Map(prev)
        next.set(meta.uuid, {
          id: meta.uuid,
          response: {
            code: opts.statusCode,
            headers: hdrs,
            delay: opts.delay,
            getBody: async () => opts.responseBody,
          },
        })

        return next
      })

      return meta.uuid
    },
    [api, db]
  )

  const delSession = useCallback(
    async (id: string, opts?: Options): Promise<void> => {
      await db.deleteSession(id) // delete session from database (FAST)

      // update state by removing the session
      setSessions((prev) => {
        const next = new Map(prev)
        next.delete(id)
        return next
      })

      // delete session from the backend (SLOW)
      await api.deleteSession(id, { signal: opts?.signal })
    },
    [api, db]
  )

  useEffect(() => {
    const ctrl = new AbortController()

    void (async (): Promise<void> => {
      // load the list of sessions (from DB)
      const ids = await db.getSessionIDs()
      if (!ids.length || ctrl.signal.aborted) {
        return
      }

      // craft a map of sessions with its details
      const current = new Map<string, Session>()
      for (const s of await Promise.all(ids.map((id) => db.getSession(id)))) {
        if (s) {
          current.set(s.id, { id: s.id, response: s.response })
        }
      }

      // no data (or unmounted) - no funny bunny honey
      if (!current.size || ctrl.signal.aborted) {
        return
      }

      // FIRST STATE UPDATE - with the data from DB (which may contain sessions that no longer exist on the backend)
      setSessions(new Map(current))

      // continue with slow operation - validate sessions against the backend
      const rm: Array<string> = []

      // clean up current list from non-existing sessions
      for (const [id, exists] of Object.entries(
        await api.checkSessionExists([...current.keys()], { signal: ctrl.signal })
      )) {
        if (!current.has(id)) {
          continue
        }

        if (!exists) {
          current.delete(id)
          rm.push(id)
        }
      }

      // if there are sessions to remove, and we're still mounted
      if (rm.length && !ctrl.signal.aborted) {
        await db.deleteSession(...rm)

        // SECOND STATE UPDATE - nonexistent sessions are removed
        setSessions(new Map(current))
      }
    })().catch((err) => {
      if (err instanceof DOMException && err.name === 'AbortError') {
        return
      }

      errHandler?.(anyToError(err))
    })

    return () => ctrl.abort()
  }, [api, db, errHandler])

  return <ctx.Provider value={{ sessions, newSession, delSession }}>{children}</ctx.Provider>
}

/** Hook that returns the sessions context. Must be called inside a SessionsProvider. */
export const useSessions = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useSessions must be used within a SessionsProvider')
  }

  return context
}

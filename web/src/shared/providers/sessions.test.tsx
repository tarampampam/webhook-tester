import React from 'react'
import { describe, expect, test, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { Client } from '~/api'
import { type Database } from '~/db'
import { newTestDatabase } from '~/test-utils'
import { SessionsProvider, useSessions } from './sessions'

const makeWrapper = (api: Client, db: Database, errHandler?: (err: Error) => void) =>
  function Wrapper({ children }: { children: React.ReactNode }): React.JSX.Element {
    return (
      <SessionsProvider api={api} db={db} errHandler={errHandler}>
        {children}
      </SessionsProvider>
    )
  }

const seedSession = async (db: Database, id: string) => {
  await db.putSession({
    id,
    response: { code: 200, headers: [] as Array<{ name: string; value: string }>, delay: 0 },
    createdAt: new Date(0),
  })
}

const apiSessionMeta = (uuid: string) => Object.freeze({ uuid, createdAt: Object.freeze(new Date(0)) })

describe('useSessions', () => {
  test('throws when called outside SessionsProvider', () => {
    expect(() => renderHook(() => useSessions())).toThrow('useSessions must be used within a SessionsProvider')
  })
})

describe('SessionsProvider', () => {
  describe('mount reconciliation', () => {
    test('prunes stale sessions on mount, keeps valid ones, and deletes pruned from DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      await seedSession(db, 'alive')
      await seedSession(db, 'stale')
      vi.spyOn(api, 'checkSessionExists').mockResolvedValue({
        alive: true,
        stale: false,
      } as Record<string, boolean>)

      const { result } = renderHook(() => useSessions(), { wrapper: makeWrapper(api, db) })

      await waitFor(() => {
        expect(result.current.sessions.has('alive')).toBe(true)
        expect(result.current.sessions.has('stale')).toBe(false)
      })
      expect(await db.getSession('stale')).toBeNull()
    })

    test('calls errHandler on backend failure but keeps DB-loaded sessions visible', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      const errHandler = vi.fn()

      await seedSession(db, 's1')
      vi.spyOn(api, 'checkSessionExists').mockRejectedValue(new Error('network error'))

      const { result } = renderHook(() => useSessions(), { wrapper: makeWrapper(api, db, errHandler) })

      await waitFor(() => expect(errHandler).toHaveBeenCalledOnce())
      expect(errHandler).toHaveBeenCalledWith(expect.objectContaining({ message: 'network error' }))
      expect(result.current.sessions.has('s1')).toBe(true)
    })
  })

  describe('newSession', () => {
    test('creates session on backend, persists to DB, and updates sessions map', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      vi.spyOn(api, 'newSession').mockResolvedValue(apiSessionMeta('new-uuid'))

      const { result } = renderHook(() => useSessions(), { wrapper: makeWrapper(api, db) })
      await act(async () => {})

      let uuid: string | undefined
      await act(async () => {
        uuid = await result.current.newSession({ statusCode: 201, delay: 100 }, {})
      })

      expect(uuid).toBe('new-uuid')
      expect(result.current.sessions.get('new-uuid')?.response.code).toBe(201)
      expect(result.current.sessions.get('new-uuid')?.response.delay).toBe(100)
      expect(await db.getSession('new-uuid')).not.toBeNull()
    })

    test('propagates api error without updating state', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      vi.spyOn(api, 'newSession').mockRejectedValue(new Error('api failure'))

      const { result } = renderHook(() => useSessions(), { wrapper: makeWrapper(api, db) })
      await act(async () => {})

      await expect(
        act(async () => {
          await result.current.newSession({}, {})
        })
      ).rejects.toThrow('api failure')

      expect(result.current.sessions.size).toBe(0)
    })
  })

  describe('delSession', () => {
    const setupWithSession = async (db: Database, api: Client) => {
      vi.spyOn(api, 'newSession').mockResolvedValue(apiSessionMeta('del-uuid'))
      const { result } = renderHook(() => useSessions(), { wrapper: makeWrapper(api, db) })
      await act(async () => {})
      await act(async () => {
        await result.current.newSession({}, {})
      })
      return result
    }

    test('removes session from state, DB, and backend', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      const deleteSpy = vi.spyOn(api, 'deleteSession').mockResolvedValue(true)

      const result = await setupWithSession(db, api)
      await act(async () => {
        await result.current.delSession('del-uuid', {})
      })

      expect(result.current.sessions.has('del-uuid')).toBe(false)
      expect(await db.getSession('del-uuid')).toBeNull()
      expect(deleteSpy).toHaveBeenCalledOnce()
    })

    test('optimistically removes from state - no rollback on api error', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      vi.spyOn(api, 'deleteSession').mockRejectedValue(new Error('backend delete failed'))

      const result = await setupWithSession(db, api)

      let caughtError: Error | null = null
      await act(async () => {
        await result.current.delSession('del-uuid', {}).catch((e: unknown) => {
          caughtError = e instanceof Error ? e : new Error(String(e))
        })
      })

      expect(caughtError).toHaveProperty('message', 'backend delete failed')
      expect(result.current.sessions.has('del-uuid')).toBe(false)
    })
  })
})

import React, { useState } from 'react'
import { describe, expect, test, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { Client, RequestEventAction } from '~/api'
import type { Database } from '~/db'
import { newTestDatabase } from '~/test-utils'
import { RequestsProvider, useRequests } from './requests'

// extract WS event shape from Client's private types
type WsHandlers = Parameters<Client['subscribeToSessionRequests']>[1]
type WsEvent = Parameters<WsHandlers['onUpdate']>[0]

const apiReq = (uuid: string, capturedAtMs: number, payload = new Uint8Array()) =>
  Object.freeze({
    uuid,
    clientAddress: '1.2.3.4',
    method: 'POST' as const,
    requestPayload: payload,
    headers: Object.freeze([]) as ReadonlyArray<{ name: string; value: string }>,
    url: Object.freeze(new URL('http://localhost/hook')),
    capturedAt: Object.freeze(new Date(capturedAtMs)),
  })

const wsCreate = (uuid: string, capturedAtMs: number): WsEvent =>
  Object.freeze({
    action: RequestEventAction.create,
    request: Object.freeze({
      uuid,
      clientAddress: '1.2.3.4',
      method: 'POST' as const,
      headers: Object.freeze([]) as ReadonlyArray<{ name: string; value: string }>,
      url: Object.freeze(new URL('http://localhost/hook')),
      capturedAt: Object.freeze(new Date(capturedAtMs)),
    }),
  })

const wsDelete = (uuid: string): WsEvent =>
  Object.freeze({
    action: RequestEventAction.delete,
    request: Object.freeze({
      uuid,
      clientAddress: '1.2.3.4',
      method: 'POST' as const,
      headers: Object.freeze([]) as ReadonlyArray<{ name: string; value: string }>,
      url: Object.freeze(new URL('http://localhost/hook')),
      capturedAt: Object.freeze(new Date(0)),
    }),
  })

const wsClear = (): WsEvent => Object.freeze({ action: RequestEventAction.clear, request: null })

const dbReq = (sID: string, rID: string, capturedAtMs: number) => ({
  sID,
  rID,
  method: 'GET',
  clientAddress: '1.2.3.4',
  url: 'http://localhost/hook',
  capturedAt: new Date(capturedAtMs),
  headers: [] as ReadonlyArray<{ name: string; value: string }>,
})

const makeWrapper = (api: Client, db: Database, requestsLimit?: number, errHandler?: (err: Error) => void) =>
  function Wrapper({ children }: { children: React.ReactNode }): React.JSX.Element {
    return (
      <RequestsProvider api={api} db={db} requestsLimit={requestsLimit} errHandler={errHandler}>
        {children}
      </RequestsProvider>
    )
  }

/** Wrapper with a runtime-mutable limit - call setLimit() to drive prop changes during a test. */
const makeDynamicWrapper = (api: Client, db: Database) => {
  let externalSet: (n: number | undefined) => void = () => {}

  const Wrapper = ({ children }: { children: React.ReactNode }): React.JSX.Element => {
    const [limit, setLimit] = useState<number | undefined>(undefined)
    externalSet = setLimit
    return (
      <RequestsProvider api={api} db={db} requestsLimit={limit}>
        {children}
      </RequestsProvider>
    )
  }

  const setLimit = (n: number | undefined): Promise<void> =>
    act(async () => {
      externalSet(n)
    })

  return { Wrapper, setLimit }
}

/** Mocks subscribeToSessionRequests and returns an emit() function to push events into the provider. */
const mockWs = (api: Client) => {
  let onUpdate: WsHandlers['onUpdate'] | null = null

  vi.spyOn(api, 'subscribeToSessionRequests').mockImplementation(async (_sID, handlers) => {
    onUpdate = handlers.onUpdate
    return (): void => {}
  })

  return {
    emit: (event: WsEvent): void => {
      onUpdate?.(event)
    },
  }
}

// ---------- tests ----------

describe('useRequests', () => {
  test('throws when called outside RequestsProvider', () => {
    expect(() => renderHook(() => useRequests())).toThrow('useRequests must be used within a RequestsProvider')
  })
})

describe('RequestsProvider', () => {
  describe('setSessionID - initial load', () => {
    test('(no limit) DB empty, API returns 3 requests - state and DB contain those 3', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('r1', 3000),
        apiReq('r2', 2000),
        apiReq('r3', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(3)
      expect([...result.current.requests.keys()]).toEqual(expect.arrayContaining(['r1', 'r2', 'r3']))

      const dbRows = await db.getRequests('s1')
      expect(dbRows.map((r) => r.rID)).toEqual(expect.arrayContaining(['r1', 'r2', 'r3']))
    })

    test('(no limit) DB has 2 stale entries not in API - stale deleted from DB, state holds 3 fresh', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      await db.putRequests([dbReq('s1', 'stale-1', 100), dbReq('s1', 'stale-2', 200)])

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('fresh-1', 3000),
        apiReq('fresh-2', 2000),
        apiReq('fresh-3', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(3)
      expect(result.current.requests.has('stale-1')).toBe(false)
      expect(result.current.requests.has('stale-2')).toBe(false)

      const dbRows = await db.getRequests('s1')
      expect(dbRows).toHaveLength(3)
      expect(dbRows.map((r) => r.rID)).not.toContain('stale-1')
    })

    test('(no limit) DB already in sync with API - state reconciled, no phantom duplicates', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      await db.putRequests([dbReq('s1', 'r1', 2000), dbReq('s1', 'r2', 1000)])
      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 2000), apiReq('r2', 1000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(2)
    })

    test('(limit=2) DB empty, API returns 4 - only 2 newest in state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('r-newest', 4000),
        apiReq('r-2nd', 3000),
        apiReq('r-3rd', 2000),
        apiReq('r-oldest', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db, 2) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(2)
      expect(result.current.requests.has('r-newest')).toBe(true)
      expect(result.current.requests.has('r-2nd')).toBe(true)
      expect(result.current.requests.has('r-3rd')).toBe(false)
      expect(result.current.requests.has('r-oldest')).toBe(false)

      const dbRows = await db.getRequests('s1')
      expect(dbRows).toHaveLength(2)
      expect(dbRows.map((r) => r.rID)).toEqual(expect.arrayContaining(['r-newest', 'r-2nd']))
    })

    test('(limit=2) DB has 3 entries matching API - only 2 newest survive in state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      await db.putRequests([dbReq('s1', 'r1', 3000), dbReq('s1', 'r2', 2000), dbReq('s1', 'r3', 1000)])
      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('r1', 3000),
        apiReq('r2', 2000),
        apiReq('r3', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db, 2) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(2)
      expect(result.current.requests.has('r1')).toBe(true)
      expect(result.current.requests.has('r2')).toBe(true)
      expect(result.current.requests.has('r3')).toBe(false)

      expect(await db.getRequests('s1')).toHaveLength(2)
    })

    test('(limit=2) DB has 1 stale entry, API returns 3 fresh - stale deleted, 2 newest from API kept', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      await db.putRequests([dbReq('s1', 'stale', 9999)])
      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('fresh-1', 3000),
        apiReq('fresh-2', 2000),
        apiReq('fresh-3', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db, 2) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(result.current.requests.size).toBe(2)
      expect(result.current.requests.has('stale')).toBe(false)
      expect(result.current.requests.has('fresh-1')).toBe(true)
      expect(result.current.requests.has('fresh-2')).toBe(true)

      const dbRows = await db.getRequests('s1')
      expect(dbRows).toHaveLength(2)
      expect(dbRows.map((r) => r.rID)).not.toContain('stale')
    })
  })

  describe('setSessionID - limit changes after load', () => {
    test('limit undefined → 3: retroactive trim applied to state and DB, 3 newest kept', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([
        apiReq('r1', 4000),
        apiReq('r2', 3000),
        apiReq('r3', 2000),
        apiReq('r4', 1000),
      ])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { Wrapper, setLimit } = makeDynamicWrapper(api, db)
      const { result } = renderHook(() => useRequests(), { wrapper: Wrapper })

      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.size).toBe(4)

      await setLimit(3)

      await waitFor(async () => {
        expect(result.current.requests.size).toBe(3)
        expect(await db.getRequests('s1')).toHaveLength(3)
      })

      expect(result.current.requests.has('r1')).toBe(true)
      expect(result.current.requests.has('r2')).toBe(true)
      expect(result.current.requests.has('r3')).toBe(true)
      expect(result.current.requests.has('r4')).toBe(false)
    })
  })

  describe('unsetSessionID and session switching', () => {
    test('unsetSessionID clears state', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.size).toBe(1)

      act(() => {
        result.current.unsetSessionID()
      })
      expect(result.current.requests.size).toBe(0)
    })

    test("switching session A → B: state shows only B's requests", async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      const getRequests = vi.spyOn(api, 'getSessionRequests')
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})

      getRequests.mockResolvedValueOnce([apiReq('a1', 1000), apiReq('a2', 2000)])
      getRequests.mockResolvedValueOnce([apiReq('b1', 3000)])

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })

      await act(async () => {
        await result.current.setSessionID('sA')
      })
      expect(result.current.requests.size).toBe(2)

      await act(async () => {
        await result.current.setSessionID('sB')
      })
      expect(result.current.requests.size).toBe(1)
      expect(result.current.requests.has('b1')).toBe(true)
      expect(result.current.requests.has('a1')).toBe(false)
    })
  })

  describe('WebSocket events', () => {
    test('create event - payload accessible via getPayload() after API fetch completes', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      const payload = new Uint8Array([1, 2, 3])

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([])
      vi.spyOn(api, 'getSessionRequest').mockResolvedValue(apiReq('ws-r1', 5000, payload))
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      await act(async () => {
        ws.emit(wsCreate('ws-r1', 5000))
      })

      await waitFor(async () => {
        const r = result.current.requests.get('ws-r1')
        expect(r).toBeDefined()
        expect(await r?.getPayload()).toEqual(payload)
      })
    })

    test('create event with limit=2, already at capacity - newest added, oldest evicted', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r-old', 1000), apiReq('r-mid', 2000)])
      vi.spyOn(api, 'getSessionRequest').mockResolvedValue(apiReq('r-new', 3000))
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db, 2) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.size).toBe(2)

      await act(async () => {
        ws.emit(wsCreate('r-new', 3000))
      })

      await waitFor(() => {
        expect(result.current.requests.has('r-new')).toBe(true)
        expect(result.current.requests.has('r-old')).toBe(false)
      })

      expect(result.current.requests.size).toBe(2)
      expect(result.current.requests.has('r-mid')).toBe(true)
    })

    test('delete event - request removed from state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000)])
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.has('r1')).toBe(true)

      await act(async () => {
        ws.emit(wsDelete('r1'))
      })
      await waitFor(() => expect(result.current.requests.has('r1')).toBe(false))

      const dbRows = await db.getRequests('s1')
      expect(dbRows.find((r) => r.rID === 'r1')).toBeUndefined()
    })

    test('clear event - all requests removed from state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000), apiReq('r2', 2000)])
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.size).toBe(2)

      await act(async () => {
        ws.emit(wsClear())
      })
      await waitFor(() => expect(result.current.requests.size).toBe(0))

      expect(await db.getRequests('s1')).toHaveLength(0)
    })

    test('events arriving after unsetSessionID are ignored', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([])
      vi.spyOn(api, 'getSessionRequest').mockResolvedValue(apiReq('late', 9000))
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      act(() => {
        result.current.unsetSessionID()
      })

      await act(async () => {
        ws.emit(wsCreate('late', 9000))
      })
      await act(async () => {}) // drain pending microtasks

      expect(result.current.requests.size).toBe(0)
    })
  })

  describe('delRequest', () => {
    test('API returns true - request removed from state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})
      vi.spyOn(api, 'deleteSessionRequest').mockResolvedValue(true)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      await act(async () => {
        await result.current.delRequest('s1', 'r1')
      })

      expect(result.current.requests.has('r1')).toBe(false)
      expect(await db.getRequests('s1')).toHaveLength(0)
    })

    test('API returns false - state and DB unchanged', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})
      vi.spyOn(api, 'deleteSessionRequest').mockResolvedValue(false)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      await act(async () => {
        await result.current.delRequest('s1', 'r1')
      })

      expect(result.current.requests.has('r1')).toBe(true)
      expect(await db.getRequests('s1')).toHaveLength(1)
    })
  })

  describe('delAllRequests', () => {
    test('API returns true - all session requests removed from state and DB', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000), apiReq('r2', 2000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})
      vi.spyOn(api, 'deleteAllSessionRequests').mockResolvedValue(true)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })
      expect(result.current.requests.size).toBe(2)

      await act(async () => {
        await result.current.delAllRequests('s1')
      })

      expect(result.current.requests.size).toBe(0)
      expect(await db.getRequests('s1')).toHaveLength(0)
    })

    test('API returns false - state and DB unchanged', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([apiReq('r1', 1000)])
      vi.spyOn(api, 'subscribeToSessionRequests').mockResolvedValue((): void => {})
      vi.spyOn(api, 'deleteAllSessionRequests').mockResolvedValue(false)

      const { result } = renderHook(() => useRequests(), { wrapper: makeWrapper(api, db) })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      await act(async () => {
        await result.current.delAllRequests('s1')
      })

      expect(result.current.requests.size).toBe(1)
    })
  })

  describe('error handling', () => {
    test('setSessionID API failure invokes errHandler', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      const errHandler = vi.fn()

      vi.spyOn(api, 'getSessionRequests').mockRejectedValue(new Error('network down'))

      const { result } = renderHook(() => useRequests(), {
        wrapper: makeWrapper(api, db, undefined, errHandler),
      })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      expect(errHandler).toHaveBeenCalledOnce()
      expect(errHandler).toHaveBeenCalledWith(expect.objectContaining({ message: 'network down' }))
    })

    test('WebSocket create - getSessionRequest failure invokes errHandler, provider stays functional', async () => {
      const api = new Client({ baseUrl: 'http://test' })
      const db = newTestDatabase()
      const errHandler = vi.fn()

      vi.spyOn(api, 'getSessionRequests').mockResolvedValue([])
      vi.spyOn(api, 'getSessionRequest').mockRejectedValue(new Error('payload fetch failed'))
      const ws = mockWs(api)

      const { result } = renderHook(() => useRequests(), {
        wrapper: makeWrapper(api, db, undefined, errHandler),
      })
      await act(async () => {
        await result.current.setSessionID('s1')
      })

      await act(async () => {
        ws.emit(wsCreate('r1', 1000))
      })
      await waitFor(() => expect(errHandler).toHaveBeenCalled())

      expect(errHandler).toHaveBeenCalledWith(expect.objectContaining({ message: 'payload fetch failed' }))
      // first setRequests (before getSessionRequest) already added the request - it stays in state
      expect(result.current.requests.has('r1')).toBe(true)
      // payload is not available since the fetch failed
      expect(await result.current.requests.get('r1')?.getPayload()).toBeNull()
    })
  })
})

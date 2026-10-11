import { describe, expect, test } from 'vitest'
import { newTestDatabase } from '~/test-utils'
import { Database } from './database'

type SessionInput = NonNullable<Parameters<Database['putSession']>[0]>
type RequestInput = Parameters<Database['putRequests']>[0][number]

const makeSession = (overrides: Partial<SessionInput> = {}): SessionInput => ({
  id: 'session-1',
  response: { code: 200, headers: [], delay: 0 },
  createdAt: new Date('2024-01-01'),
  ...overrides,
})

const makeRequest = (overrides: Partial<RequestInput> = {}): RequestInput => ({
  sID: 'session-1',
  rID: 'request-1',
  clientAddress: '127.0.0.1',
  method: 'POST',
  headers: [],
  url: '/webhook',
  capturedAt: new Date('2024-01-01T10:00:00'),
  ...overrides,
})

describe('putSession / getSession', () => {
  test('stores and retrieves metadata by id', async () => {
    const db = newTestDatabase()
    const s = makeSession()
    await db.putSession(s)
    const record = await db.getSession(s.id)
    expect(record?.id).toBe(s.id)
    expect(record?.response.code).toBe(s.response.code)
    expect(record?.response.headers).toEqual(s.response.headers)
    expect(record?.response.delay).toBe(s.response.delay)
  })

  test('getBody returns stored body', async () => {
    const db = newTestDatabase()
    const body = new Uint8Array([1, 2, 3])
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body } }))
    const record = await db.getSession('session-1')
    expect(await record?.response.getBody()).toEqual(body)
  })

  test('getBody returns null when no body stored', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession())
    const record = await db.getSession('session-1')
    expect(await record?.response.getBody()).toBeNull()
  })

  test('empty body is stored — getBody returns empty Uint8Array', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array() } }))
    const record = await db.getSession('session-1')
    expect(await record?.response.getBody()).toEqual(new Uint8Array())
  })

  test('upsert replaces existing session metadata', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0 } }))
    await db.putSession(makeSession({ response: { code: 418, headers: [], delay: 0 } }))
    expect((await db.getSession('session-1'))?.response.code).toBe(418)
  })

  test('upsert replaces existing session body', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array([1]) } }))
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array([2]) } }))
    expect(await (await db.getSession('session-1'))?.response.getBody()).toEqual(new Uint8Array([2]))
  })

  test('upsert with empty body overwrites existing body', async () => {
    const db = newTestDatabase()
    await db.putSession(
      makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array([1, 2, 3]) } })
    )
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array() } }))
    const record = await db.getSession('session-1')
    expect(await record?.response.getBody()).toEqual(new Uint8Array())
  })

  test('returns null for unknown id', async () => {
    expect(await newTestDatabase().getSession('nope')).toBeNull()
  })

  test('no-op for empty args', async () => {
    await expect(newTestDatabase().putSession()).resolves.toBeUndefined()
  })
})

describe('getSessionIDs', () => {
  test('returns [] when DB is empty', async () => {
    expect(await newTestDatabase().getSessionIDs()).toEqual([])
  })

  test('orders IDs newest first by createdAt', async () => {
    const db = newTestDatabase()
    await db.putSession(
      makeSession({ id: 'old', createdAt: new Date('2024-01-01') }),
      makeSession({ id: 'new', createdAt: new Date('2024-01-03') }),
      makeSession({ id: 'mid', createdAt: new Date('2024-01-02') })
    )
    expect(await db.getSessionIDs()).toEqual(['new', 'mid', 'old'])
  })
})

describe('deleteSession', () => {
  test('removes the session', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession())
    await db.deleteSession('session-1')
    expect(await db.getSession('session-1')).toBeNull()
  })

  test('deletes associated session body', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession({ response: { code: 200, headers: [], delay: 0, body: new Uint8Array([1]) } }))
    await db.deleteSession('session-1')
    // re-insert metadata only; body from old session must not survive
    await db.putSession(makeSession())
    const record = await db.getSession('session-1')
    expect(await record?.response.getBody()).toBeNull()
  })

  test('cascades to all associated requests', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession())
    await db.putRequests([makeRequest({ rID: 'r1' }), makeRequest({ rID: 'r2' })])
    await db.deleteSession('session-1')
    expect(await db.getRequest('r1')).toBeNull()
    expect(await db.getRequest('r2')).toBeNull()
  })

  test('cascades to request payloads', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession())
    await db.putRequests([makeRequest({ rID: 'r1', payload: new Uint8Array([9]) })])
    await db.deleteSession('session-1')
    // re-insert request without payload; old payload must not survive
    await db.putSession(makeSession())
    await db.putRequests([makeRequest({ rID: 'r1' })])
    expect(await (await db.getRequest('r1'))?.getPayload()).toBeNull()
  })

  test('does not remove requests belonging to other sessions', async () => {
    const db = newTestDatabase()
    await db.putSession(makeSession({ id: 's1' }), makeSession({ id: 's2' }))
    await db.putRequests([makeRequest({ sID: 's1', rID: 'r1' }), makeRequest({ sID: 's2', rID: 'r2' })])
    await db.deleteSession('s1')
    expect(await db.getRequest('r2')).not.toBeNull()
  })

  test('no-op for empty args', async () => {
    await expect(newTestDatabase().deleteSession()).resolves.toBeUndefined()
  })
})

describe('putRequests / getRequest', () => {
  test('stores and retrieves metadata by rID', async () => {
    const db = newTestDatabase()
    const r = makeRequest()
    await db.putRequests([r])
    const record = await db.getRequest(r.rID)
    expect(record?.rID).toBe(r.rID)
    expect(record?.method).toBe(r.method)
    expect(record?.url).toBe(r.url)
  })

  test('getPayload returns stored payload', async () => {
    const db = newTestDatabase()
    const payload = new Uint8Array([4, 5, 6])
    await db.putRequests([makeRequest({ payload })])
    const record = await db.getRequest('request-1')
    expect(await record?.getPayload()).toEqual(payload)
  })

  test('getPayload returns null when no payload stored', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest()])
    const record = await db.getRequest('request-1')
    expect(await record?.getPayload()).toBeNull()
  })

  test('empty payload is stored — getPayload returns empty Uint8Array', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ payload: new Uint8Array() })])
    expect(await (await db.getRequest('request-1'))?.getPayload()).toEqual(new Uint8Array())
  })

  test('upsert replaces existing request with same rID', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ method: 'GET' })])
    await db.putRequests([makeRequest({ method: 'DELETE' })])
    expect((await db.getRequest('request-1'))?.method).toBe('DELETE')
  })

  test('upsert without payload deletes existing payload', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ payload: new Uint8Array([7, 8]) })])
    await db.putRequests([makeRequest()])
    expect(await (await db.getRequest('request-1'))?.getPayload()).toBeNull()
  })

  test('returns null for unknown rID', async () => {
    expect(await newTestDatabase().getRequest('nope')).toBeNull()
  })

  test('no-op for empty array', async () => {
    await expect(newTestDatabase().putRequests([])).resolves.toBeUndefined()
  })
})

describe('putRequests limit', () => {
  test('trims oldest requests when count exceeds limit', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
      makeRequest({ rID: 'r3', capturedAt: new Date('2024-01-01T12:00:00') }),
    ])
    await db.putRequests([makeRequest({ rID: 'r4', capturedAt: new Date('2024-01-01T13:00:00') })], 2)
    const ids = (await db.getRequests('session-1')).map((r) => r.rID)
    expect(ids).toEqual(['r4', 'r3'])
  })

  test('deletes payloads for trimmed requests', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00'), payload: new Uint8Array([1]) }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00'), payload: new Uint8Array([2]) }),
      makeRequest({ rID: 'r3', capturedAt: new Date('2024-01-01T12:00:00'), payload: new Uint8Array([3]) }),
    ])
    await db.putRequests([makeRequest({ rID: 'r4', capturedAt: new Date('2024-01-01T13:00:00') })], 2)
    // r1 trimmed - its payload must be gone
    await db.putRequests([makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') })])
    expect(await (await db.getRequest('r1'))?.getPayload()).toBeNull()
  })

  test('trims independently per sID', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ sID: 's1', rID: 's1r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ sID: 's1', rID: 's1r2', capturedAt: new Date('2024-01-01T11:00:00') }),
      makeRequest({ sID: 's2', rID: 's2r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ sID: 's2', rID: 's2r2', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    await db.putRequests(
      [
        makeRequest({ sID: 's1', rID: 's1r3', capturedAt: new Date('2024-01-01T12:00:00') }),
        makeRequest({ sID: 's2', rID: 's2r3', capturedAt: new Date('2024-01-01T12:00:00') }),
      ],
      2
    )
    const s1ids = (await db.getRequests('s1')).map((r) => r.rID)
    const s2ids = (await db.getRequests('s2')).map((r) => r.rID)
    expect(s1ids).toEqual(['s1r3', 's1r2'])
    expect(s2ids).toEqual(['s2r3', 's2r2'])
  })

  test('no trimming when count is within limit', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    await db.putRequests([makeRequest({ rID: 'r3', capturedAt: new Date('2024-01-01T12:00:00') })], 5)
    const ids = (await db.getRequests('session-1')).map((r) => r.rID)
    expect(ids).toEqual(['r3', 'r2', 'r1'])
  })

  test('no trimming when limit is 0', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
      makeRequest({ rID: 'r3', capturedAt: new Date('2024-01-01T12:00:00') }),
    ])
    await db.putRequests([makeRequest({ rID: 'r4', capturedAt: new Date('2024-01-01T13:00:00') })], 0)
    expect(await db.getRequests('session-1')).toHaveLength(4)
  })
})

describe('trimRequests', () => {
  test('trims oldest requests keeping the most recent N', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
      makeRequest({ rID: 'r3', capturedAt: new Date('2024-01-01T12:00:00') }),
    ])
    await db.trimRequests('session-1', 2)
    const ids = (await db.getRequests('session-1')).map((r) => r.rID)
    expect(ids).toEqual(['r3', 'r2'])
  })

  test('deletes payloads for trimmed requests', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00'), payload: new Uint8Array([1]) }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    await db.trimRequests('session-1', 1)
    // r1 trimmed - its payload must be gone
    await db.putRequests([makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') })])
    expect(await (await db.getRequest('r1'))?.getPayload()).toBeNull()
  })

  test('does not affect requests for other sessions', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ sID: 's1', rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ sID: 's1', rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
      makeRequest({ sID: 's2', rID: 'r3', capturedAt: new Date('2024-01-01T10:00:00') }),
    ])
    await db.trimRequests('s1', 1)
    expect(await db.getRequests('s2')).toHaveLength(1)
  })

  test('no trimming when count is within limit', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    await db.trimRequests('session-1', 5)
    expect(await db.getRequests('session-1')).toHaveLength(2)
  })

  test('no-op when limit is 0', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'r2', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    await db.trimRequests('session-1', 0)
    expect(await db.getRequests('session-1')).toHaveLength(2)
  })
})

describe('putRequestPayload', () => {
  test('stores a payload that is then readable via getRequest', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1' })])
    const payload = new Uint8Array([10, 20, 30])
    await db.putRequestPayload('r1', payload)
    expect(await (await db.getRequest('r1'))?.getPayload()).toEqual(payload)
  })

  test('upserts — replaces an existing payload', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1', payload: new Uint8Array([1]) })])
    const updated = new Uint8Array([2, 3])
    await db.putRequestPayload('r1', updated)
    expect(await (await db.getRequest('r1'))?.getPayload()).toEqual(updated)
  })

  test('newRequestPayloadReader reflects the payload written after the reader was created', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1' })])
    const read = db.newRequestPayloadReader('r1')
    expect(await read()).toBeNull()
    const payload = new Uint8Array([7, 8, 9])
    await db.putRequestPayload('r1', payload)
    expect(await read()).toEqual(payload)
  })
})

describe('getRequests', () => {
  test('returns [] for unknown sID', async () => {
    expect(await newTestDatabase().getRequests('nope')).toEqual([])
  })

  test('returns only requests for the given sID', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ sID: 's1', rID: 'r1' }), makeRequest({ sID: 's2', rID: 'r2' })])
    const result = await db.getRequests('s1')
    expect(result).toHaveLength(1)
    expect(result[0]?.rID).toBe('r1')
  })

  test('orders by capturedAt descending', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'old', capturedAt: new Date('2024-01-01T10:00:00') }),
      makeRequest({ rID: 'new', capturedAt: new Date('2024-01-01T12:00:00') }),
      makeRequest({ rID: 'mid', capturedAt: new Date('2024-01-01T11:00:00') }),
    ])
    const ids = (await db.getRequests('session-1')).map((r) => r.rID)
    expect(ids).toEqual(['new', 'mid', 'old'])
  })

  test('getPayload returns payload on results from getRequests', async () => {
    const db = newTestDatabase()
    const payload = new Uint8Array([1, 2, 3])
    await db.putRequests([makeRequest({ rID: 'r1', payload })])
    const [result] = await db.getRequests('session-1')
    expect(await result?.getPayload()).toEqual(payload)
  })

  test('getPayload returns null on results from getRequests when no payload stored', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1' })])
    const [result] = await db.getRequests('session-1')
    expect(await result?.getPayload()).toBeNull()
  })
})

describe('deleteRequest', () => {
  test('removes specified requests', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1' }), makeRequest({ rID: 'r2' })])
    await db.deleteRequest('r1')
    expect(await db.getRequest('r1')).toBeNull()
    expect(await db.getRequest('r2')).not.toBeNull()
  })

  test('deletes associated payload', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1', payload: new Uint8Array([1]) })])
    await db.deleteRequest('r1')
    // re-insert without payload; old payload must not survive
    await db.putRequests([makeRequest({ rID: 'r1' })])
    expect(await (await db.getRequest('r1'))?.getPayload()).toBeNull()
  })

  test('no-op for empty args', async () => {
    await expect(newTestDatabase().deleteRequest()).resolves.toBeUndefined()
  })
})

describe('deleteAllRequests', () => {
  test('removes all requests for the given sID', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ rID: 'r1' }), makeRequest({ rID: 'r2' })])
    await db.deleteAllRequests('session-1')
    expect(await db.getRequest('r1')).toBeNull()
    expect(await db.getRequest('r2')).toBeNull()
  })

  test('deletes associated payloads', async () => {
    const db = newTestDatabase()
    await db.putRequests([
      makeRequest({ rID: 'r1', payload: new Uint8Array([1]) }),
      makeRequest({ rID: 'r2', payload: new Uint8Array([2]) }),
    ])
    await db.deleteAllRequests('session-1')
    // re-insert without payloads; old payloads must not survive
    await db.putRequests([makeRequest({ rID: 'r1' }), makeRequest({ rID: 'r2' })])
    expect(await (await db.getRequest('r1'))?.getPayload()).toBeNull()
    expect(await (await db.getRequest('r2'))?.getPayload()).toBeNull()
  })

  test('does not affect requests for other sessions', async () => {
    const db = newTestDatabase()
    await db.putRequests([makeRequest({ sID: 's1', rID: 'r1' }), makeRequest({ sID: 's2', rID: 'r2' })])
    await db.deleteAllRequests('s1')
    expect(await db.getRequest('r2')).not.toBeNull()
  })
})

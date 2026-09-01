import { describe, expect, test, vi } from 'vitest'
import { Client } from './client'
import { APIErrorCommon, APIErrorNotFound } from './errors'

const BASE_URL = 'http://unit-test'

/** Creates a fetch mock that returns a single JSON response. */
const jsonFetch = (status: number, body: unknown) =>
  vi
    .fn()
    .mockResolvedValue(new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }))

/** Creates a fetch mock that captures request bodies and returns a fixed JSON response. */
const captureFetch = (status: number, body: unknown) => {
  const bodies: Array<unknown> = []
  const response = new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
  const mock = vi.fn().mockImplementation(async (req: Request) => {
    bodies.push(await req.json())
    return response
  })
  return { mock, bodies }
}

const SESSION_FIXTURE = {
  uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
  response: {
    status_code: 201,
    headers: [{ name: 'X-Foo', value: 'bar' }],
    delay: 3,
    response_body_base64: 'aGVsbG8=', // "hello"
  },
  created_at_unix_milli: 1_600_000_000_000,
}

const CREATE_SESSION_FIXTURE = {
  uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
  created_at_unix_milli: 1_600_000_000_000,
}

const REQUEST_FIXTURE = {
  uuid: 'ffffffff-0000-1111-2222-333333333333',
  client_address: '1.2.3.4',
  method: 'POST',
  request_payload_base64: 'aGVsbG8=', // "hello"
  headers: [{ name: 'Content-Type', value: 'text/plain' }],
  url: 'http://localhost/test-path',
  captured_at_unix_milli: 2_000_000_000,
}

/** @see {@link Client.currentVersion} */
describe('currentVersion', () => {
  test('strips leading lowercase v', async () => {
    expect(await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(200, { version: 'v1.2.3' }) }).currentVersion()).toBe(
      '1.2.3'
    )
  })

  test('strips leading uppercase V', async () => {
    expect(await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(200, { version: 'V3.2.1' }) }).currentVersion()).toBe(
      '3.2.1'
    )
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(new Client({ baseUrl: BASE_URL, fetch: jsonFetch(503, {}) }).currentVersion()).rejects.toBeInstanceOf(
      APIErrorCommon
    )
  })
})

/** @see {@link Client.latestVersion} */
describe('latestVersion', () => {
  test('strips leading lowercase v', async () => {
    expect(await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(200, { version: 'v2.0.0' }) }).latestVersion()).toBe(
      '2.0.0'
    )
  })

  test('strips leading uppercase V', async () => {
    expect(await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(200, { version: 'V2.0.0' }) }).latestVersion()).toBe(
      '2.0.0'
    )
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(new Client({ baseUrl: BASE_URL, fetch: jsonFetch(503, {}) }).latestVersion()).rejects.toBeInstanceOf(
      APIErrorCommon
    )
  })
})

/** @see {@link Client.getSettings} */
describe('getSettings', () => {
  test('maps all fields', async () => {
    const s = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, {
        limits: { max_requests: 128, max_request_body_size: 1024, session_ttl: 3600 },
        tunnel: { enabled: true, url: 'https://tunnel.example.com' },
        public_url_root: 'https://example.com/api',
      }),
    }).getSettings()

    expect(s.limits.maxRequests).toBe(128)
    expect(s.limits.maxRequestBodySize).toBe(1024)
    expect(s.limits.sessionTTL).toBe(3600)
    expect(s.tunnel.enabled).toBe(true)
    expect(s.tunnel.url).toBeInstanceOf(URL)
    expect(s.tunnel.url?.href).toBe('https://tunnel.example.com/')
    expect(s.publicUrlRoot).toBeInstanceOf(URL)
    expect(s.publicUrlRoot?.href).toBe('https://example.com/api')
  })

  test('tunnel.url is null when absent', async () => {
    const s = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, {
        limits: { max_requests: 10, max_request_body_size: 512, session_ttl: 60 },
        tunnel: { enabled: false },
      }),
    }).getSettings()

    expect(s.tunnel.url).toBeNull()
    expect(s.publicUrlRoot).toBeNull()
  })

  test('strips trailing slashes from publicUrlRoot', async () => {
    const s = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, {
        limits: { max_requests: 10, max_request_body_size: 512, session_ttl: 60 },
        tunnel: { enabled: false },
        public_url_root: 'https://example.com/api//',
      }),
    }).getSettings()

    expect(s.publicUrlRoot?.href).toBe('https://example.com/api')
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(new Client({ baseUrl: BASE_URL, fetch: jsonFetch(500, {}) }).getSettings()).rejects.toBeInstanceOf(
      APIErrorCommon
    )
  })
})

/** @see {@link Client.newSession} */
describe('newSession', () => {
  test('maps all response fields', async () => {
    const s = await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(201, CREATE_SESSION_FIXTURE) }).newSession({})

    expect(s.uuid).toBe(CREATE_SESSION_FIXTURE.uuid)
    expect(s.createdAt).toBeInstanceOf(Date)
    expect(s.createdAt.getTime()).toBe(1_600_000_000_000)
  })

  test('clamps statusCode above 530 to 530', async () => {
    const { mock, bodies } = captureFetch(201, CREATE_SESSION_FIXTURE)
    await new Client({ baseUrl: BASE_URL, fetch: mock }).newSession({ statusCode: 9999 })

    expect(bodies[0]).toMatchObject({ status_code: 530 })
  })

  test('clamps statusCode below 100 to 100', async () => {
    const { mock, bodies } = captureFetch(201, CREATE_SESSION_FIXTURE)
    await new Client({ baseUrl: BASE_URL, fetch: mock }).newSession({ statusCode: 1 })

    expect(bodies[0]).toMatchObject({ status_code: 100 })
  })

  test('clamps delay above 30 to 30', async () => {
    const { mock, bodies } = captureFetch(201, CREATE_SESSION_FIXTURE)
    await new Client({ baseUrl: BASE_URL, fetch: mock }).newSession({ delay: 999 })

    expect(bodies[0]).toMatchObject({ delay: 30 })
  })

  test('strips headers with empty values', async () => {
    const { mock, bodies } = captureFetch(201, CREATE_SESSION_FIXTURE)
    await new Client({ baseUrl: BASE_URL, fetch: mock }).newSession({
      headers: { 'X-Keep': 'value', 'X-Drop': '' },
    })

    expect(bodies[0]).toMatchObject({
      headers: expect.arrayContaining([{ name: 'X-Keep', value: 'value' }]),
    })
    expect(bodies[0]).toMatchObject({
      headers: expect.not.arrayContaining([expect.objectContaining({ name: 'X-Drop' })]),
    })
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(new Client({ baseUrl: BASE_URL, fetch: jsonFetch(503, {}) }).newSession({})).rejects.toBeInstanceOf(
      APIErrorCommon
    )
  })
})

/** @see {@link Client.getSession} */
describe('getSession', () => {
  test('maps response fields', async () => {
    const s = await new Client({ baseUrl: BASE_URL, fetch: jsonFetch(200, SESSION_FIXTURE) }).getSession('any-id')

    expect(s.uuid).toBe(SESSION_FIXTURE.uuid)
    expect(s.response.statusCode).toBe(201)
    expect(s.createdAt).toBeInstanceOf(Date)
    expect(s.createdAt.getTime()).toBe(1_600_000_000_000)
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).getSession('bad-id')
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })
})

/** @see {@link Client.checkSessionExists} */
describe('checkSessionExists', () => {
  test('true for existing, false for explicitly-false and missing IDs', async () => {
    const existing = 'aaaa-1111'
    const explicitlyFalse = 'bbbb-2222'
    const missing = 'cccc-3333'

    const result = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, { [existing]: true, [explicitlyFalse]: false }),
    }).checkSessionExists([existing, explicitlyFalse, missing])

    expect(result[existing]).toBe(true)
    expect(result[explicitlyFalse]).toBe(false)
    expect(result[missing]).toBe(false)
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(500, {}) }).checkSessionExists(['any'])
    ).rejects.toBeInstanceOf(APIErrorCommon)
  })
})

/** @see {@link Client.deleteSession} */
describe('deleteSession', () => {
  test.each([true, false])('returns data.success = %s', async (success) => {
    const result = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, { success }),
    }).deleteSession('any-id')

    expect(result).toBe(success)
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).deleteSession('missing-id')
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(500, {}) }).deleteSession('any-id')
    ).rejects.toBeInstanceOf(APIErrorCommon)
  })
})

/** @see {@link Client.getSessionRequests} */
describe('getSessionRequests', () => {
  test('maps all fields', async () => {
    const reqs = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, [REQUEST_FIXTURE]),
    }).getSessionRequests('session-id')

    expect(reqs).toHaveLength(1)

    const [r] = reqs
    if (!r) {
      throw new Error('expected at least one request')
    }
    expect(r.uuid).toBe(REQUEST_FIXTURE.uuid)
    expect(r.clientAddress).toBe('1.2.3.4')
    expect(r.method).toBe('POST')
    expect(r.requestPayload).toEqual(new TextEncoder().encode('hello'))
    expect(r.headers).toEqual([{ name: 'Content-Type', value: 'text/plain' }])
    expect(r.url).toBeInstanceOf(URL)
    expect(r.url.pathname).toBe('/test-path')
    expect(r.capturedAt).toBeInstanceOf(Date)
    expect(r.capturedAt.getTime()).toBe(2_000_000_000)
  })

  test('sorts requests newest first', async () => {
    const older = { ...REQUEST_FIXTURE, uuid: 'older', captured_at_unix_milli: 1_000_000 }
    const newer = { ...REQUEST_FIXTURE, uuid: 'newer', captured_at_unix_milli: 2_000_000 }

    const reqs = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, [older, newer]),
    }).getSessionRequests('session-id')

    expect(reqs).toHaveLength(2)
    expect(reqs[0]?.uuid).toBe('newer')
    expect(reqs[1]?.uuid).toBe('older')
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).getSessionRequests('bad-id')
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })
})

/** @see {@link Client.deleteAllSessionRequests} */
describe('deleteAllSessionRequests', () => {
  test.each([true, false])('returns data.success = %s', async (success) => {
    const result = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, { success }),
    }).deleteAllSessionRequests('any-id')

    expect(result).toBe(success)
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).deleteAllSessionRequests(
        'missing-id'
      )
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(500, {}) }).deleteAllSessionRequests('any-id')
    ).rejects.toBeInstanceOf(APIErrorCommon)
  })
})

/** @see {@link Client.getSessionRequest} */
describe('getSessionRequest', () => {
  test('maps all fields', async () => {
    const r = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, REQUEST_FIXTURE),
    }).getSessionRequest('session-id', 'request-id')

    expect(r.uuid).toBe(REQUEST_FIXTURE.uuid)
    expect(r.clientAddress).toBe('1.2.3.4')
    expect(r.method).toBe('POST')
    expect(r.requestPayload).toEqual(new TextEncoder().encode('hello'))
    expect(r.headers).toEqual([{ name: 'Content-Type', value: 'text/plain' }])
    expect(r.url).toBeInstanceOf(URL)
    expect(r.capturedAt).toBeInstanceOf(Date)
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).getSessionRequest('s', 'r')
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })
})

/** @see {@link Client.deleteSessionRequest} */
describe('deleteSessionRequest', () => {
  test.each([true, false])('returns data.success = %s', async (success) => {
    const result = await new Client({
      baseUrl: BASE_URL,
      fetch: jsonFetch(200, { success }),
    }).deleteSessionRequest('session-id', 'request-id')

    expect(result).toBe(success)
  })

  test('throws APIErrorNotFound on 404', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(404, { error: 'not found' }) }).deleteSessionRequest(
        's',
        'missing-r'
      )
    ).rejects.toBeInstanceOf(APIErrorNotFound)
  })

  test('throws APIErrorCommon on non-ok response', async () => {
    await expect(
      new Client({ baseUrl: BASE_URL, fetch: jsonFetch(500, {}) }).deleteSessionRequest('s', 'r')
    ).rejects.toBeInstanceOf(APIErrorCommon)
  })
})

/** @see {@link Client.subscribeToSessionRequests} */
describe('subscribeToSessionRequests', () => {
  test('rejects immediately when signal is already aborted', async () => {
    const ac = new AbortController()
    ac.abort()

    await expect(
      new Client({ baseUrl: BASE_URL }).subscribeToSessionRequests(
        'session-id',
        { onUpdate: () => {} },
        { signal: ac.signal }
      )
    ).rejects.toBeInstanceOf(DOMException)
  })
})

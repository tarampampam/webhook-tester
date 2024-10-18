import { afterEach, beforeAll, describe, expect, test, vi } from 'vitest'
import { throwIfNotJSON, throwIfNotValidResponse } from './middleware'
import createFetchMock from 'vitest-fetch-mock'
import type { Middleware } from 'openapi-fetch'
import createClient from 'openapi-fetch'
import { APIErrorCommon, APIErrorNotFound } from './errors'

const fetchMocker = createFetchMock(vi)

beforeAll(() => fetchMocker.enableMocks())
afterEach(() => fetchMocker.resetMocks())

interface paths {
  '/self': {
    get: {
      responses: {
        200: {
          headers: Record<string, string>
          content: {
            'application/json': string
          }
        }
      }
    }
  }
}

const newClient = (...mv: Middleware[]) => {
  const client = createClient<paths>({ baseUrl: 'http://localhost' })

  client.use(...mv) // attach the middleware

  return client
}

describe('throwIfNotJSON', () => {
  test('pass', async () => {
    const client = newClient(throwIfNotJSON)

    fetchMocker.mockResponseOnce(() => ({
      status: 200,
      body: '"ok"',
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    const { data, error } = await client.GET('/self')

    expect(data).equals('ok')
    expect(error).toBeUndefined()
  })

  test('throws', async () => {
    const client = newClient(throwIfNotJSON)

    fetchMocker.mockResponseOnce(() => ({
      status: 200,
      body: '"ok"',
      headers: { 'Content-Type': 'text/html' }, // the header is incorrect
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false) // fail the test if the error is not thrown
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorCommon)
      expect((e as TypeError).message).toBe('Response is not JSON (the header "Content-Type" does not include "json")')
    }
  })
})

describe('throwIfNotValidResponse', () => {
  test('pass', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 200,
      body: '"ok"',
      headers: { 'Content-Type': 'text/html' }, // the header doesn't matter
    }))

    const { data, error } = await client.GET('/self')

    expect(data).equals('ok')
    expect(error).toBeUndefined()
  })

  test('throws ({ message: "..." })', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 404,
      body: `{"message": "some value"}`,
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorNotFound)
      expect((e as TypeError).message).toBe('some value')
    }
  })

  test('throws ({ error: "..." })', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 404,
      body: `{"error": "some value"}`,
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorNotFound)
      expect((e as TypeError).message).toBe('some value')
    }
  })

  test('throws ({ errors: ["...", ...] })', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 404,
      body: `{"errors": ["some", "value"]}`,
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorNotFound)
      expect((e as TypeError).message).toBe('some, value')
    }
  })

  test('throws ({ errors: { "...": "..." } })', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 500,
      body: `{"errors": {"a": "some", "b": "value"}}`,
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorCommon)
      expect((e as TypeError).message).toBe('some, value')
    }
  })

  test('throws', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 500,
      body: `{"error]`, // since the content type, the error message will be `res.statusText`
      headers: { 'Content-Type': 'foo/bar' }, // the header is incorrect
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorCommon)
      expect((e as TypeError).message).toBe('Internal Server Error')
    }
  })

  test('throws json (Failed to parse the response body as JSON)', async () => {
    const client = newClient(throwIfNotValidResponse)

    fetchMocker.mockResponseOnce(() => ({
      status: 500,
      body: `{"error]`,
      headers: { 'Content-Type': 'application/json' }, // the header is correct
    }))

    try {
      await client.GET('/self')

      expect(true).toBe(false)
    } catch (e: TypeError | unknown) {
      expect(e).toBeInstanceOf(APIErrorCommon)
      expect((e as TypeError).message).toBe('Internal Server Error (failed to parse the response body as JSON)')
    }
  })
})

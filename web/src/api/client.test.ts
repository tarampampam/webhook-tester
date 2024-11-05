import { afterAll, beforeAll, describe, expect, test } from 'vitest'
import fetchMock from '@fetch-mock/vitest'
import { APIErrorCommon } from './errors'
import { Client } from './client'

beforeAll(() => fetchMock.mockGlobal())
afterAll(() => fetchMock.mockRestore())

const baseUrl = 'http://localhost'

/** A quick way to test if an error is thrown is to check it. */
const expectError = async (fn: () => Promise<unknown> | unknown, checkErrorFn: (e: TypeError) => void) => {
  try {
    await fn()

    expect(true).toBe(false) // fail the test if the error is not thrown
  } catch (e: TypeError | unknown) {
    expect(e).toBeInstanceOf(Error)

    checkErrorFn(e as Error)
  }
}

describe('currentVersion', () => {
  const mockUrlMatcher = /\/api\/version$/

  test('pass', async () => {
    fetchMock.getOnce(mockUrlMatcher, { status: 200, body: { version: 'v1.2.3' } })

    const client = new Client({ baseUrl })

    expect((await client.currentVersion()).toString()).equals('1.2.3')
    expect((await client.currentVersion()).toString()).equals('1.2.3') // the second call should use the cache

    fetchMock.getOnce(mockUrlMatcher, { status: 200, body: { version: 'V3.2.1' } })

    expect((await client.currentVersion(true)).toString()).equals('3.2.1') // the cache should be updated
    expect((await client.currentVersion()).toString()).equals('3.2.1') // the second call should use the cache
  })

  test('throws', async () => {
    fetchMock.getOnce(mockUrlMatcher, { status: 501, body: '"error"' })

    await expectError(
      async () => await new Client({ baseUrl }).currentVersion(),
      (e) => {
        expect(e).toBeInstanceOf(APIErrorCommon)
        expect(e.message).toBe('Not Implemented')
      }
    )
  })
})

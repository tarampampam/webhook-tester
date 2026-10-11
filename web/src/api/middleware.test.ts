import { APIErrorBadRequest, APIErrorNotFound } from './errors'
import { throwIfBadRequest, throwIfNotFound } from './middleware'
import { describe, expect, test } from 'vitest'

function callOnResponse(middleware: typeof throwIfBadRequest, response: Response) {
  const { onResponse } = middleware
  if (typeof onResponse !== 'function') {
    throw new Error('middleware.onResponse is not defined')
  }
  return onResponse({ response } as Parameters<typeof onResponse>[0])
}

function makeResponse(status: number, body?: BodyInit | null, contentType?: string): Response {
  return new Response(body, {
    status,
    headers: contentType ? { 'Content-Type': contentType } : {},
  })
}

describe('throwIfBadRequest', () => {
  test('throws APIErrorBadRequest with message from JSON body', async () => {
    const response = makeResponse(400, JSON.stringify({ message: 'Validation failed' }), 'application/json')

    await expect(callOnResponse(throwIfBadRequest, response)).rejects.toSatisfy(
      (e: unknown) => e instanceof APIErrorBadRequest && e.message === 'Validation failed' && e.response === response
    )
  })

  test('throws APIErrorBadRequest with default message when no content-type', async () => {
    const response = makeResponse(400)

    await expect(callOnResponse(throwIfBadRequest, response)).rejects.toSatisfy(
      (e: unknown) => e instanceof APIErrorBadRequest && e.message === 'Bad request (400)'
    )
  })

  test.each([200, 401, 403, 404, 500])('does not throw for status %d', async (status) => {
    const response = makeResponse(status)
    await expect(callOnResponse(throwIfBadRequest, response)).resolves.toBeUndefined()
  })
})

describe('throwIfNotFound', () => {
  test('throws APIErrorNotFound with message from JSON body', async () => {
    const response = makeResponse(404, JSON.stringify({ message: 'Session not found' }), 'application/json')

    await expect(callOnResponse(throwIfNotFound, response)).rejects.toSatisfy(
      (e: unknown) => e instanceof APIErrorNotFound && e.message === 'Session not found' && e.response === response
    )
  })

  test('throws APIErrorNotFound with default message when no content-type', async () => {
    const response = makeResponse(404)

    await expect(callOnResponse(throwIfNotFound, response)).rejects.toSatisfy(
      (e: unknown) => e instanceof APIErrorNotFound && e.message === 'Not found (404)'
    )
  })

  test.each([200, 400, 500])('does not throw for status %d', async (status) => {
    const response = makeResponse(status)
    await expect(callOnResponse(throwIfNotFound, response)).resolves.toBeUndefined()
  })
})

import { extractErrorMessage } from './utils'
import { describe, expect, test } from 'vitest'

describe('extractErrorMessage', () => {
  const makeResponse = (status: number, body?: BodyInit | null, contentType?: string): Response => {
    return new Response(body, {
      status,
      headers: contentType ? { 'Content-Type': contentType } : {},
    })
  }

  describe('ok response', () => {
    test.each([200, 201, 299])('returns null for status %d regardless of body', async (status) => {
      const response = makeResponse(status, JSON.stringify({ message: 'some message' }), 'application/json')
      expect(await extractErrorMessage(response)).toBeNull()
    })
  })

  describe('missing or unsupported Content-Type', () => {
    test('returns null when Content-Type header is absent', async () => {
      const response = makeResponse(400, null)
      expect(await extractErrorMessage(response)).toBeNull()
    })

    test('returns null for unsupported content type', async () => {
      const response = makeResponse(400, 'data', 'application/octet-stream')
      expect(await extractErrorMessage(response)).toBeNull()
    })
  })

  describe('text/* content type', () => {
    test('returns body text for text/plain', async () => {
      const response = makeResponse(400, 'Bad input', 'text/plain')
      expect(await extractErrorMessage(response)).toBe('Bad input')
    })

    test('returns body text for text/html', async () => {
      const response = makeResponse(500, '<h1>Error</h1>', 'text/html')
      expect(await extractErrorMessage(response)).toBe('<h1>Error</h1>')
    })
  })

  describe('JSON content type', () => {
    test('returns string body as-is', async () => {
      const response = makeResponse(400, JSON.stringify('Validation error'), 'application/json')
      expect(await extractErrorMessage(response)).toBe('Validation error')
    })

    test('throws when JSON body is malformed', async () => {
      const response = makeResponse(400, '{invalid}', 'application/json')

      await expect(extractErrorMessage(response)).rejects.toThrow('Failed to parse the response body as JSON')
    })

    test.each([
      ['null body', null, null],
      ['empty array', [], null],
      ['empty object', {}, null],
      ['number body', 42, null],
      ['boolean body', true, null],
    ])('returns null for %s', async (_label, body, expected) => {
      const response = makeResponse(400, JSON.stringify(body), 'application/json')
      expect(await extractErrorMessage(response)).toBe(expected)
    })

    describe('object with known error fields', () => {
      test('extracts message from { message }', async () => {
        const response = makeResponse(400, JSON.stringify({ message: 'Validation failed' }), 'application/json')
        expect(await extractErrorMessage(response)).toBe('Validation failed')
      })

      test('extracts error from { error }', async () => {
        const response = makeResponse(400, JSON.stringify({ error: 'Unauthorized' }), 'application/json')
        expect(await extractErrorMessage(response)).toBe('Unauthorized')
      })

      test('extracts and joins string items from { errors: string[] }', async () => {
        const response = makeResponse(
          400,
          JSON.stringify({ errors: ['Field is required', 'Invalid format'] }),
          'application/json'
        )
        expect(await extractErrorMessage(response)).toBe('Field is required, Invalid format')
      })

      test('filters out non-string items from { errors: mixed[] }', async () => {
        const response = makeResponse(
          400,
          JSON.stringify({ errors: ['Field is required', 42, null, 'Invalid format'] }),
          'application/json'
        )
        expect(await extractErrorMessage(response)).toBe('Field is required, Invalid format')
      })

      test('returns null for { errors: [] }', async () => {
        const response = makeResponse(400, JSON.stringify({ errors: [] }), 'application/json')
        expect(await extractErrorMessage(response)).toBeNull()
      })

      test('extracts and joins values from { errors: Record<string, string> }', async () => {
        const response = makeResponse(
          400,
          JSON.stringify({ errors: { name: 'Required', email: 'Invalid' } }),
          'application/json'
        )
        expect(await extractErrorMessage(response)).toBe('Required, Invalid')
      })

      test('falls back to JSON.stringify for unrecognized object shape', async () => {
        const body = { foo: 'bar', count: 1 }
        const response = makeResponse(400, JSON.stringify(body), 'application/json')
        expect(await extractErrorMessage(response)).toBe(JSON.stringify(body))
      })

      test('message takes precedence over error field', async () => {
        const response = makeResponse(
          400,
          JSON.stringify({ message: 'From message', error: 'From error' }),
          'application/json'
        )
        expect(await extractErrorMessage(response)).toBe('From message')
      })
    })

    test('handles vendor JSON content type (application/vnd.api+json)', async () => {
      const response = makeResponse(400, JSON.stringify({ message: 'Not found' }), 'application/vnd.api+json')
      expect(await extractErrorMessage(response)).toBe('Not found')
    })
  })
})

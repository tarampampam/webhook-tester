import { anyToError, throwIfAny } from './errors'
import { describe, expect, test } from 'vitest'

describe('throwIfAny', () => {
  test.each([
    ['an Error instance', new Error('boom')],
    ['a non-empty string', 'something went wrong'],
    ['a truthy number', 42],
    ['a plain object', { message: 'oops' }],
  ])('throws when given %s', (_label, value) => {
    expect(() => throwIfAny(value)).toThrow()
  })

  test.each([
    ['null', null],
    ['undefined', undefined],
    ['an empty string', ''],
    ['zero', 0],
    ['false', false],
  ])('does not throw when given %s', (_label, value) => {
    expect(() => throwIfAny(value)).not.toThrow()
  })

  test('rethrows an Error instance preserving identity', () => {
    const err = new Error('original')
    expect(() => throwIfAny(err)).toThrowError(err)
  })

  test('throws an Error instance even when given a raw string', () => {
    expect(() => throwIfAny('raw string')).toThrowError(Error)
  })
})

describe('anyToError', () => {
  describe('given an Error instance', () => {
    test('returns the same object (identity)', () => {
      const err = new Error('original')
      expect(anyToError(err)).toBe(err)
    })

    test('preserves Error subclass instances', () => {
      const err = new TypeError('bad type')
      expect(anyToError(err)).toBe(err)
    })
  })

  describe('given a string', () => {
    test.each([
      ['non-empty string', 'something went wrong', 'something went wrong'],
      ['empty string', '', ''],
    ])('wraps %s into an Error with the same message', (_label, input, expectedMessage) => {
      const result = anyToError(input)
      expect(result).toBeInstanceOf(Error)
      expect(result.message).toBe(expectedMessage)
    })
  })

  describe('given null or undefined', () => {
    test.each([
      ['null', null],
      ['undefined', undefined],
    ])('returns an Error with "Unknown error" for %s', (_label, value) => {
      const result = anyToError(value)
      expect(result).toBeInstanceOf(Error)
      expect(result.message).toBe('Unknown error')
    })
  })

  describe('given a regular object (has Object.prototype)', () => {
    test('uses the message property when it is a string', () => {
      const result = anyToError({ message: 'from message' })
      expect(result).toBeInstanceOf(Error)
      expect(result.message).toBe('from message')
    })

    test('message takes priority over a custom toString()', () => {
      const result = anyToError({ message: 'msg wins', toString: () => 'toString result' })
      expect(result.message).toBe('msg wins')
    })

    test('falls back to toString() when there is no message property', () => {
      const obj = { toString: () => 'custom repr' }
      expect(anyToError(obj).message).toBe('custom repr')
    })

    test('uses Object.prototype.toString() for bare objects without message', () => {
      // Regular objects always satisfy `'toString' in obj`, so they never
      // reach the cause / JSON.stringify branches.
      expect(anyToError({ value: 42 }).message).toBe('[object Object]')
    })
  })

  describe('given a null-prototype object (no inherited toString)', () => {
    test('uses a string cause property', () => {
      const obj: Record<string, unknown> = Object.create(null)
      Object.assign(obj, { cause: 'cause message' })
      expect(anyToError(obj).message).toBe('cause message')
    })

    test('returns the Error cause directly', () => {
      const cause = new Error('the cause')
      const obj: Record<string, unknown> = Object.create(null)
      Object.assign(obj, { cause })
      expect(anyToError(obj)).toBe(cause)
    })

    test('falls back to JSON.stringify when no recognized property is present', () => {
      const obj: Record<string, unknown> = Object.create(null)
      Object.assign(obj, { foo: 'bar', n: 1 })
      expect(anyToError(obj).message).toBe('{"foo":"bar","n":1}')
    })
  })

  describe('given other primitives', () => {
    test.each([
      ['a positive number', 42, '42'],
      ['a negative number', -1, '-1'],
      ['true', true, 'true'],
      ['false', false, 'false'],
    ])('converts %s via String() into an Error message', (_label, input, expectedMessage) => {
      const result = anyToError(input)
      expect(result).toBeInstanceOf(Error)
      expect(result.message).toBe(expectedMessage)
    })
  })
})

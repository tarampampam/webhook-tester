import { describe, expect, test } from 'vitest'
import { methodToColor } from './color'

describe('methodToColor', () => {
  test.each([
    ['GET', 'green'],
    ['POST', 'yellow'],
    ['PUT', 'blue'],
    ['PATCH', 'violet'],
    ['DELETE', 'red'],
    ['HEAD', 'green'],
    ['OPTIONS', 'orange'],
    ['TRACE', 'pink'],
    ['CONNECT', 'indigo'],
  ])('%s -> %s', (method, expected) => {
    expect(methodToColor(method)).toBe(expected)
  })

  test('returns gray for unknown method', () => {
    expect(methodToColor('UNKNOWN')).toBe('gray')
    expect(methodToColor('FOO')).toBe('gray')
    expect(methodToColor('')).toBe('gray')
  })

  test('is case-insensitive', () => {
    expect(methodToColor('get')).toBe('green')
    expect(methodToColor('Post')).toBe('yellow')
    expect(methodToColor('dElEtE')).toBe('red')
  })

  test('trims surrounding whitespace', () => {
    expect(methodToColor('  GET  ')).toBe('green')
    expect(methodToColor('\tPOST\n')).toBe('yellow')
  })
})

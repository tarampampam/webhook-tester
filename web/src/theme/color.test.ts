import { describe, expect, test } from 'vitest'
import { methodToColor, statusCodeToColor } from './color'

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

describe('statusCodeToColor', () => {
  test.each([
    [100, 'teal'],
    [200, 'teal'],
    [201, 'teal'],
    [299, 'teal'],
    [300, 'orange'],
    [301, 'orange'],
    [399, 'orange'],
    [400, 'red'],
    [404, 'red'],
    [499, 'red'],
    [500, 'cyan'],
    [502, 'cyan'],
    [599, 'cyan'],
  ])('%i -> %s', (code, expected) => {
    expect(statusCodeToColor(code)).toBe(expected)
  })
})

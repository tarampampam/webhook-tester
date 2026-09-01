import { describe, expect, test } from 'vitest'
import { base64ToUint8Array, uint8ArrayToBase64 } from './encoding'

describe('base64ToUint8Array', () => {
  test.each([
    ['empty string', '', new Uint8Array([])],
    ['null byte', 'AA==', new Uint8Array([0])],
    ['0xFF byte', '/w==', new Uint8Array([255])],
    ['+ character (index 62)', '++++', new Uint8Array([251, 239, 190])],
    ['three bytes, no padding', 'AAEC', new Uint8Array([0, 1, 2])],
    ['"hello"', 'aGVsbG8=', new Uint8Array([104, 101, 108, 108, 111])],
    ['"test"', 'dGVzdA==', new Uint8Array([116, 101, 115, 116])],
    ['7500 zero bytes', 'A'.repeat(10000), new Uint8Array(7500).fill(0)],
    ['7500 0xFF bytes', '/'.repeat(10000), new Uint8Array(7500).fill(255)],
  ])('decodes %s correctly', (_label, b64, expected) => {
    expect(base64ToUint8Array(b64)).toEqual(expected)
  })

  test.each(['not!valid', '!@#$', 'abc!'])('throws for invalid base64 %s', (b64) => {
    expect(() => base64ToUint8Array(b64)).toThrow()
  })
})

describe('uint8ArrayToBase64', () => {
  test.each([
    ['empty array', new Uint8Array([]), ''],
    ['null byte', new Uint8Array([0]), 'AA=='],
    ['0xFF byte', new Uint8Array([255]), '/w=='],
    ['+ character (index 62)', new Uint8Array([251, 239, 190]), '++++'],
    ['three bytes, no padding needed', new Uint8Array([0, 1, 2]), 'AAEC'],
    ['"hello"', new Uint8Array([104, 101, 108, 108, 111]), 'aGVsbG8='],
    ['"test"', new Uint8Array([116, 101, 115, 116]), 'dGVzdA=='],
    ['300 zero bytes', new Uint8Array(300).fill(0), 'A'.repeat(400)],
    ['300 0xFF bytes', new Uint8Array(300).fill(255), '/'.repeat(400)],
  ])('encodes %s to correct base64', (_label, bytes, expected) => {
    expect(uint8ArrayToBase64(bytes)).toBe(expected)
  })

  test('full byte range 0–255 roundtrips through base64ToUint8Array', () => {
    const bytes = new Uint8Array(256)
    for (let i = 0; i < 256; i++) {
      bytes[i] = i
    }
    expect(base64ToUint8Array(uint8ArrayToBase64(bytes))).toEqual(bytes)
  })
})

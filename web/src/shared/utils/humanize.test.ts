import { humanizeBytes, humanizeDuration } from './humanize'
import { describe, expect, test } from 'vitest'

describe('humanizeBytes', () => {
  test('returns "0 B" for zero', () => {
    expect(humanizeBytes(0)).toBe('0 B')
  })

  test('throws RangeError for negative input', () => {
    expect(() => humanizeBytes(-1)).toThrow(RangeError)
  })

  test.each([
    [1, '1 B'],
    [512, '512 B'],
    [1023, '1023 B'],
  ])('formats %i bytes without unit conversion', (input, expected) => {
    expect(humanizeBytes(input)).toBe(expected)
  })

  test.each([
    [1024, '1.00 KiB'],
    [1536, '1.50 KiB'],
    [1048576, '1.00 MiB'],
    [1073741824, '1.00 GiB'],
    [1099511627776, '1.00 TiB'],
  ])('converts %i bytes to %s', (input, expected) => {
    expect(humanizeBytes(input)).toBe(expected)
  })
})

describe('humanizeDuration', () => {
  test.each([
    [5025, '1:23:45'],
    [3600, '1:00:00'],
    [7384, '2:03:04'],
  ])('formats %ds with hours as "%s"', (input, expected) => {
    expect(humanizeDuration(input)).toBe(expected)
  })

  test.each([
    [145, '02:25'],
    [60, '01:00'],
    [0, '00:00'],
    [59, '00:59'],
  ])('formats %ds without hours as "%s"', (input, expected) => {
    expect(humanizeDuration(input)).toBe(expected)
  })

  test('pads minutes and seconds with leading zeros', () => {
    expect(humanizeDuration(61)).toBe('01:01')
  })

  test.each([
    [145.9, '02:25.90'],
    [59.99, '00:59.99'],
    [3600.5, '1:00:00.50'],
  ])('fractional seconds: %ds → "%s"', (input, expected) => {
    expect(humanizeDuration(input)).toBe(expected)
  })

  test.each([
    [60, '01:00'],
    [145.9, '02:25'],
    [59.99, '00:59'],
    [3600.5, '1:00:00'],
  ])('fractional seconds without "withMS" param: %ds → "%s"', (input, expected) => {
    expect(humanizeDuration(input, false)).toBe(expected)
  })

  test.each([
    [-1, '-00:01'],
    [-0.5, '-00:00.50'],
    [-145, '-02:25'],
    [-3600.5, '-1:00:00.50'],
  ])('negative values: %ds → "%s"', (input, expected) => {
    expect(humanizeDuration(input)).toBe(expected)
  })
})

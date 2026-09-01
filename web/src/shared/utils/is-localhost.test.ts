import { describe, expect, test } from 'vitest'
import { isLocalhost } from './is-localhost'

describe('isLocalhost', () => {
  test.each([
    // localhost names
    { host: 'localhost', expected: true },
    { host: 'localhost.localdomain', expected: true },
    { host: 'ip6-localhost', expected: true },
    { host: 'ip6-loopback', expected: true },
    { host: 'LOCALHOST', expected: true }, // case-insensitive
    { host: '  localhost  ', expected: true }, // trims whitespace

    // IPv4 loopback (127/8)
    { host: '127.0.0.1', expected: true },
    { host: '127.255.255.255', expected: true }, // top of 127/8
    { host: '127.999.0.1', expected: false }, // invalid octet
    { host: '127.0.0.256', expected: false }, // invalid octet
    { host: '128.0.0.1', expected: false }, // outside 127/8

    // 0.0.0.0 / unspecified
    { host: '0.0.0.0', expected: true },

    // IPv6 loopback — bare (window.location.hostname format)
    { host: '::1', expected: true },
    { host: '0:0:0:0:0:0:0:1', expected: true }, // full-form loopback
    { host: '::', expected: true }, // IPv6 unspecified (like 0.0.0.0)
    { host: '0:0:0:0:0:0:0:0', expected: true }, // full-form unspecified
    { host: '::2', expected: false },

    // IPv6 — bracket-enclosed (window.location.host format)
    { host: '[::1]', expected: true },
    { host: '[::1]:443', expected: true }, // with port
    { host: '[0:0:0:0:0:0:0:1]:8080', expected: true },

    // IPv4-mapped IPv6
    { host: '::ffff:127.0.0.1', expected: true },
    { host: '::ffff:127.255.255.255', expected: true }, // top of ::ffff:127/8
    { host: '::ffff:7f00:1', expected: true }, // hex notation of 127.0.0.1
    { host: '::ffff:192.168.1.1', expected: false },

    // with port
    { host: 'localhost:8080', expected: true },
    { host: '127.0.0.1:3000', expected: true },

    // non-localhost
    { host: 'example.com', expected: false },
    { host: '192.168.1.1', expected: false },
    { host: '10.0.0.1', expected: false },
    { host: '8.8.8.8', expected: false },
    { host: '2001:db8::1', expected: false },
  ])('$host -> $expected', ({ host, expected }) => {
    expect(isLocalhost(host)).toBe(expected)
  })
})

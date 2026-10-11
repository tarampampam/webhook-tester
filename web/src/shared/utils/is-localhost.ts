const LOCALHOST_NAMES = new Set(['localhost', 'localhost.localdomain', 'ip6-localhost', 'ip6-loopback'])

const IPV6_LOOPBACKS = new Set(['::1', '0:0:0:0:0:0:0:1', '::ffff:127.0.0.1', '::ffff:7f00:1', '::', '0:0:0:0:0:0:0:0'])

/**
 * Checks if the given host is a localhost address.
 */
export const isLocalhost = (host: string): boolean => {
  // Strip port if present (handle IPv6 brackets too)
  const hostname = host
    .replace(/^\[(.+?)](:\d+)?$/, '$1') // [::1] or [::1]:8080
    .replace(/^([^:]+)(:\d+)$/, '$1') // hostname:port or IPv4:port
    .toLowerCase()
    .trim()

  if (LOCALHOST_NAMES.has(hostname)) {
    return true
  }

  // IPv4 loopback: 127.0.0.0/8
  if (
    /^127\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)$/.test(
      hostname
    )
  ) {
    return true
  }

  // 0.0.0.0
  if (hostname === '0.0.0.0') {
    return true
  }

  if (IPV6_LOOPBACKS.has(hostname)) {
    return true
  }

  // ::ffff:127.x.x.x (IPv4-mapped IPv6, entire 127/8 block)
  return /^::ffff:(127\.\d{1,3}\.\d{1,3}\.\d{1,3})$/.test(hostname)
}

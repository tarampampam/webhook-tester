import type { Request } from '../shared'

const METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS', 'HEAD'] as const

/**
 * Generates a new Request object with random values for testing purposes.
 */
export const newRequest = (overrides?: Partial<Request>): Request => {
  const ipV4 = Array.from({ length: 4 }, () => Math.floor(Math.random() * 256)).join('.')
  const ipv6 = Array.from({ length: 8 }, () => Math.floor(Math.random() * 65536).toString(16)).join(':')

  return {
    sID: crypto.randomUUID(),
    id: crypto.randomUUID(),
    clientAddress: Math.random() < 0.5 ? ipV4 : ipv6,
    method: METHODS[Math.floor(Math.random() * METHODS.length)] || 'GET',
    headers: [
      {
        name: 'Content-Type',
        value: 'application/json',
      },
      {
        name: 'Authorization',
        value: 'Bearer ' + crypto.randomUUID(),
      },
      {
        name: 'X-Random',
        value: crypto.randomUUID(),
      },
    ],
    url: new URL(`https://example.com/${crypto.randomUUID()}`),
    getPayload: () => Promise.resolve(new TextEncoder().encode(JSON.stringify({ message: 'Hello, world!' }))),
    capturedAt: new Date(),
    ...overrides,
  }
}

/**
 * Convert a base64 string to a Uint8Array.
 *
 * @throws {Error} If the input is not valid base64.
 */
export function base64ToUint8Array(base64: string): Uint8Array {
  let binaryString: string

  try {
    binaryString = atob(base64)
  } catch {
    throw new Error('base64ToUint8Array: invalid base64 input')
  }

  const len = binaryString.length
  const bytes = new Uint8Array(len)

  for (let i = 0; i < len; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }

  return bytes
}

/**
 * Convert a Uint8Array to a base64 string.
 */
export function uint8ArrayToBase64(uint8Array: Uint8Array): string {
  let binary = ''
  const CHUNK_SIZE = 128

  for (let i = 0; i < uint8Array.length; i += CHUNK_SIZE) {
    binary += String.fromCharCode(...uint8Array.subarray(i, i + CHUNK_SIZE))
  }

  return btoa(binary)
}

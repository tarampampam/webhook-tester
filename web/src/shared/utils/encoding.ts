/**
 * Convert a base64 string to a Uint8Array.
 */
export function base64ToUint8Array(base64: string): Uint8Array {
  const binaryString = atob(base64)
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
  let binaryString = ''

  for (const byte of uint8Array) {
    binaryString += String.fromCharCode(byte)
  }

  return btoa(binaryString)
}

const FILE_SIZE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB'] as const

/**
 * Converts a number of bytes into a human-readable string with appropriate units.
 *
 * @example
 * ```ts
 * humanizeBytes(0) // "0 B"
 * humanizeBytes(512) // "512 B"
 * humanizeBytes(1024) // "1.00 KiB"
 * humanizeBytes(1536) // "1.50 KiB"
 * humanizeBytes(1048576) // "1.00 MiB"
 * humanizeBytes(1073741824) // "1.00 GiB"
 * ```
 */
export const humanizeBytes = (bytes: number): string => {
  if (bytes < 0) {
    throw new RangeError(`Bytes must be a non-negative integer, but got ${bytes}`)
  }

  if (bytes === 0) {
    return `0 ${FILE_SIZE_UNITS[0]}`
  }

  if (bytes < 1024) {
    return `${bytes} B`
  }

  const exponent = Math.min(Math.floor(Math.log2(bytes) / 10), FILE_SIZE_UNITS.length - 1)
  const value = bytes / 1024 ** exponent

  return `${value.toFixed(2)} ${FILE_SIZE_UNITS[exponent]}`
}

/**
 * Convert seconds to a human-readable format (e.g., "1:23:45" or "23:45").
 *
 * @example
 * ```ts
 * humanizeDuration(60) // "01:00"
 * humanizeDuration(5025) // "1:23:45"
 * humanizeDuration(145.9) // "02:25.90"
 * humanizeDuration(145.9, false) // "02:25"
 * humanizeDuration(-3600.5) // "-1:00:00.50"
 * ```
 */
export const humanizeDuration = (seconds: number, withMs: boolean = true): string => {
  const abs = Math.abs(seconds)
  const h = Math.floor(abs / 3600)
  const m = Math.floor((abs % 3600) / 60)
  const s = Math.floor(abs % 60)
  const u = Math.floor((abs - Math.floor(abs)) * 100)

  const mm = String(m).padStart(2, '0')
  const ss = String(s).padStart(2, '0')
  const uu = String(u).padStart(2, '0')

  return `${seconds < 0 ? '-' : ''}${h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`}${u > 0 && withMs ? `.${uu}` : ''}`
}

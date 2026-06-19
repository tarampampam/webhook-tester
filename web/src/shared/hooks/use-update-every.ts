import { useEffect, useLayoutEffect, useRef, useState } from 'react'

/**
 * Calls `fn` on an interval of `ms` milliseconds and returns the latest result.
 *
 * Non-positive or non-finite ms disables the interval entirely.
 *
 * @example
 * ```tsx
 * const now = useUpdateEvery(() => new Date(), 100)
 *
 * return <div>{now.toLocaleTimeString()}</div>
 * ```
 */
export const useUpdateEvery = <T>(fn: () => T, ms: number): T => {
  const [value, setValue] = useState<T>(fn)
  const fnRef = useRef(fn)

  // intentionally no deps - runs after every render to keep fnRef in sync with the latest fn
  useLayoutEffect(() => {
    fnRef.current = fn
  })

  // drive periodic recalculation (restarts when ms changes)
  useEffect(() => {
    if (ms <= 0 || !isFinite(ms)) {
      return
    }

    const id = setInterval(() => {
      setValue((prev) => {
        const next = fnRef.current()
        return Object.is(prev, next) ? prev : next
      })
    }, ms)

    return () => clearInterval(id)
  }, [ms])

  return value
}

import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { useUpdateEvery } from './use-update-every'

/** @vitest-environment happy-dom */
describe('useUpdateEvery', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  test('returns the initial value from fn on mount', () => {
    const fn = vi.fn(() => 42)
    const { result } = renderHook(() => useUpdateEvery(fn, 1_000))
    expect(result.current).toBe(42)
  })

  test('updates value each time the interval fires', () => {
    let tick = 0
    const fn = vi.fn(() => tick)
    const { result } = renderHook(() => useUpdateEvery(fn, 1_000))
    expect(result.current).toBe(0)

    tick = 1
    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(result.current).toBe(1)

    tick = 2
    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(result.current).toBe(2)
  })

  test.each([0, -1, -Infinity, Infinity, NaN])('does not start an interval when ms = %s', (ms) => {
    let tick = 0
    const fn = vi.fn(() => tick)
    const { result } = renderHook(() => useUpdateEvery(fn, ms))

    tick = 99
    act(() => {
      vi.advanceTimersByTime(10_000)
    })
    expect(result.current).toBe(0)
  })

  test('picks up the latest fn on the next tick after fn reference changes', () => {
    const fnA = vi.fn(() => 'a')
    const fnB = vi.fn(() => 'b')

    const { result, rerender } = renderHook(({ fn }: { fn: () => string }) => useUpdateEvery(fn, 1_000), {
      initialProps: { fn: fnA },
    })
    expect(result.current).toBe('a')

    rerender({ fn: fnB })
    expect(result.current).toBe('a') // no immediate update

    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(result.current).toBe('b') // updated on next tick using the new fn
  })

  test('preserves state reference when fn returns the same object', () => {
    const obj = { x: 1 }
    const fn = vi.fn(() => obj)
    const { result } = renderHook(() => useUpdateEvery(fn, 100))

    expect(result.current).toBe(obj)
    act(() => {
      vi.advanceTimersByTime(100)
    })
    expect(result.current).toBe(obj)
  })

  test('stops updating after unmount', () => {
    let tick = 0
    const fn = vi.fn(() => tick)
    const { result, unmount } = renderHook(() => useUpdateEvery(fn, 100))

    unmount()
    tick = 99
    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(result.current).toBe(0)
  })

  test('restarts interval when ms changes', () => {
    let tick = 0
    const fn = vi.fn(() => tick)
    const { result, rerender } = renderHook(({ ms }) => useUpdateEvery(fn, ms), {
      initialProps: { ms: 1_000 },
    })

    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(result.current).toBe(0)

    tick = 1
    rerender({ ms: 200 })
    act(() => {
      vi.advanceTimersByTime(200)
    })
    expect(result.current).toBe(1)
  })
})

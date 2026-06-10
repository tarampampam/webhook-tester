import React from 'react'
import { beforeEach, describe, expect, test } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { LastUsedProvider, useLastUsed } from './last-used'

const wrapper = ({ children }: { children: React.ReactNode }): React.JSX.Element => (
  <LastUsedProvider>{children}</LastUsedProvider>
)

describe('useLastUsed', () => {
  test('throws when called outside LastUsedProvider', () => {
    expect(() => renderHook(() => useLastUsed())).toThrow('useLastUsed must be used within a LastUsedProvider')
  })
})

describe('LastUsedProvider', () => {
  beforeEach(() => localStorage.clear())

  test('exposes null as initial value when storage is empty', () => {
    const { result } = renderHook(() => useLastUsed(), { wrapper })

    expect(result.current.lastUsedSID).toBeNull()
  })

  test('setLastUsedSID updates the value', async () => {
    const { result } = renderHook(() => useLastUsed(), { wrapper })

    await act(async () => {
      result.current.setLastUsedSID('session-abc')
    })

    expect(result.current.lastUsedSID).toBe('session-abc')
  })

  test('setLastUsedSID(null) clears the value', async () => {
    const { result } = renderHook(() => useLastUsed(), { wrapper })

    await act(async () => {
      result.current.setLastUsedSID('session-abc')
    })
    await act(async () => {
      result.current.setLastUsedSID(null)
    })

    expect(result.current.lastUsedSID).toBeNull()
  })

  test('value is restored from localStorage on remount', async () => {
    const { result, unmount } = renderHook(() => useLastUsed(), { wrapper })

    await act(async () => {
      result.current.setLastUsedSID('persisted-id')
    })
    unmount()

    const { result: result2 } = renderHook(() => useLastUsed(), { wrapper })

    expect(result2.current.lastUsedSID).toBe('persisted-id')
  })
})

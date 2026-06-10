/** @vitest-environment happy-dom */
import { useStorage } from './use-storage'
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, test } from 'vitest'

const PREFIX = 'wht-v3-'

const dispatchStorageEvent = (storageArea: Storage, key: string | null, newValue: string | null = null) => {
  window.dispatchEvent(new StorageEvent('storage', { key, newValue, storageArea }))
}

beforeEach(() => {
  sessionStorage.clear()
  localStorage.clear()
})

describe('useStorage', () => {
  describe('initial value', () => {
    test('returns initValue when storage is empty', () => {
      const { result } = renderHook(() => useStorage('default', 'key'))

      expect(result.current[0]).toBe('default')
    })

    test.each([['session', () => sessionStorage] as const, ['local', () => localStorage] as const])(
      'reads a pre-stored value from %s storage',
      (area, getStore) => {
        getStore().setItem(`${PREFIX}key`, JSON.stringify('pre-stored'))

        const { result } = renderHook(() => useStorage('default', 'key', area))

        expect(result.current[0]).toBe('pre-stored')
      }
    )
  })

  describe('storage key prefix', () => {
    test(`uses "${PREFIX}" prefix by default`, () => {
      const { result } = renderHook(() => useStorage('v', 'mykey'))

      act(() => result.current[1]('v'))

      expect(sessionStorage.getItem(`${PREFIX}mykey`)).not.toBeNull()
    })

    test('uses custom prefix when provided', () => {
      const { result } = renderHook(() => useStorage('v', 'mykey', 'session', 'app-'))

      act(() => result.current[1]('v'))

      expect(sessionStorage.getItem('app-mykey')).not.toBeNull()
    })
  })

  describe('setter behaviour', () => {
    test('persists the value to storage and updates state', () => {
      const { result } = renderHook(() => useStorage<string | null>(null, 'key'))

      act(() => result.current[1]('hello'))

      expect(result.current[0]).toBe('hello')
      expect(sessionStorage.getItem(`${PREFIX}key`)).toBe('"hello"')
    })

    test('removes the entry from storage and sets state to null when called with null', () => {
      sessionStorage.setItem(`${PREFIX}key`, JSON.stringify('existing'))

      const { result } = renderHook(() => useStorage('existing', 'key'))

      act(() => result.current[1](null))

      expect(result.current[0]).toBeNull()
      expect(sessionStorage.getItem(`${PREFIX}key`)).toBeNull()
    })

    test('supports an updater function as the argument', () => {
      const { result } = renderHook(() => useStorage(1, 'counter'))

      act(() => result.current[1]((prev) => (prev ?? 0) + 1))

      expect(result.current[0]).toBe(2)
      expect(sessionStorage.getItem(`${PREFIX}counter`)).toBe('2')
    })

    test.each([
      ['string', 'text', '"text"'],
      ['number', 42, '42'],
      ['object', { a: 1 }, '{"a":1}'],
      ['array', [1, 2], '[1,2]'],
      ['boolean', true, 'true'],
    ] as const)('serialises %s values as JSON', (_label, value, expected) => {
      const { result } = renderHook(() => useStorage<unknown>(null, 'key'))

      act(() => result.current[1](value))

      expect(sessionStorage.getItem(`${PREFIX}key`)).toBe(expected)
    })
  })

  describe('corrupted storage value', () => {
    test('removes corrupted JSON entry and falls back to initValue', () => {
      sessionStorage.setItem(`${PREFIX}key`, 'not-valid-json{{{')

      const { result } = renderHook(() => useStorage('fallback', 'key'))

      expect(result.current[0]).toBe('fallback')
      expect(sessionStorage.getItem(`${PREFIX}key`)).toBeNull()
    })
  })

  describe('cross-tab sync via storage event', () => {
    test('updates state when another tab writes to the same key', () => {
      const { result } = renderHook(() => useStorage('initial', 'key'))

      act(() => {
        sessionStorage.setItem(`${PREFIX}key`, JSON.stringify('from-other-tab'))
        dispatchStorageEvent(sessionStorage, `${PREFIX}key`, JSON.stringify('from-other-tab'))
      })

      expect(result.current[0]).toBe('from-other-tab')
    })

    test('sets state to null when another tab clears the whole storage (key === null)', () => {
      sessionStorage.setItem(`${PREFIX}key`, JSON.stringify('something'))

      const { result } = renderHook(() => useStorage('something', 'key'))

      act(() => dispatchStorageEvent(sessionStorage, null))

      expect(result.current[0]).toBeNull()
    })

    test('ignores storage events from a different storage area', () => {
      const { result } = renderHook(() => useStorage('initial', 'key', 'session'))

      act(() => {
        localStorage.setItem(`${PREFIX}key`, JSON.stringify('wrong-area'))
        dispatchStorageEvent(localStorage, `${PREFIX}key`, JSON.stringify('wrong-area'))
      })

      expect(result.current[0]).toBe('initial')
    })
  })

  describe('same-tab sync via custom events', () => {
    test('two hook instances sharing the same key stay in sync', () => {
      const { result: a } = renderHook(() => useStorage('init', 'shared'))
      const { result: b } = renderHook(() => useStorage('init', 'shared'))

      act(() => a.current[1]('synced'))

      expect(b.current[0]).toBe('synced')
    })

    test('hook instances using different keys do not interfere', () => {
      const { result: a } = renderHook(() => useStorage('a-init', 'key-a'))
      const { result: b } = renderHook(() => useStorage('b-init', 'key-b'))

      act(() => a.current[1]('a-updated'))

      expect(b.current[0]).toBe('b-init')
    })
  })
})

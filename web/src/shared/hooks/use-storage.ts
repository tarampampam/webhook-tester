import { type Dispatch, type SetStateAction, useCallback, useEffect, useRef, useState } from 'react'

type StorageArea = 'local' | 'session'

/**
 * The cache keeps track of whether each storage type is available.
 *
 * - `undefined` -> not yet checked
 * - `null` -> checked and found unavailable
 * - `Storage` -> valid and usable
 */
const cache: { [key in StorageArea]: Storage | null | undefined } = {
  local: undefined,
  session: undefined,
}

/**
 * Retrieves the specified web storage (localStorage or sessionStorage), validating its availability and caching
 * the result for future calls.
 */
const getStorage = (area: StorageArea): Storage | null => {
  // check if we already have a cached result for this storage type
  const cached = cache[area]
  if (cached !== undefined) {
    return cached // return cached result
  }

  // if running outside the browser (e.g., in Node.js, Jest), window is not defined - storage is unavailable
  if (typeof window === 'undefined') {
    cache[area] = null // unavailable due to no window

    return null
  }

  let storage: Storage | undefined = undefined

  // select the correct storage based on the requested area
  switch (area) {
    case 'local':
      storage = window.localStorage
      break
    case 'session':
      storage = window.sessionStorage
      break
    default:
      return null // unknown area
  }

  // if storage is somehow not defined, mark it as unavailable
  if (storage === undefined) {
    cache[area] = null // unavailable due to no storage

    return null
  }

  try {
    // attempt to perform a simple write/remove operation to confirm that the storage is available
    const testKey = `__storage_test_${Math.random().toString(36).slice(2)}__`

    storage.setItem(testKey, testKey)
    storage.removeItem(testKey)

    cache[area] = storage // cache the successful check

    return storage
  } catch {
    // if any error occurs (e.g., quota exceeded, storage disabled, private mode), mark this storage as
    // unavailable to avoid future retries
    cache[area] = null

    return null
  }
}

/**
 * Custom React hook to synchronize a state variable with browser storage (localStorage or sessionStorage).
 *
 * Key Features (behavioral summary):
 *
 * - Accepts an initial value and a storage key.
 * - Serializes values to JSON when storing and parses JSON when reading.
 * - If the value is set to `null`, the hook removes the entry from storage.
 * - Synchronizes across browser tabs/windows using the native `storage` event.
 * - Synchronizes within the same tab using a custom DOM `CustomEvent` so that different hook instances
 *   receive updates immediately.
 *
 * The hook intentionally stores `null` as "no value stored".
 */
export const useStorage = <T>(
  initValue: T,
  key: string,
  area: StorageArea = 'session',
  prefix: string = 'wht-v3-'
): readonly [Readonly<T | null>, Dispatch<SetStateAction<T | null>>] => {
  const storage = getStorage(area)
  const storageKey = prefix + key
  const eventName = `storage-hook:${area}:${storageKey}:updated`

  // unique identifier for this instance (hook) to ignore its own events
  const [senderId] = useState(() => Math.random().toString(36).slice(2))
  const senderIdRef = useRef(senderId)
  type StorageNotifyEvent = CustomEvent<{ senderId: string }>

  /** Get the current value from storage, or null if not available. */
  const get = useCallback((): T | null => {
    if (!storage) {
      return null
    }

    const loaded: string | null = storage.getItem(storageKey)

    try {
      return loaded !== null ? JSON.parse(loaded) : null
    } catch {
      storage.removeItem(storageKey) // corrupted value - remove it to avoid future errors

      return null
    }
  }, [storage, storageKey])

  /** Set a new value in storage, or remove it if the value is null. */
  const set = useCallback(
    (value: T | null): void => {
      if (!storage) {
        return
      }

      if (value === null) {
        storage.removeItem(storageKey)
      } else {
        storage.setItem(storageKey, JSON.stringify(value))
      }
    },
    [storage, storageKey]
  )

  /** Remove the value from storage. */
  const remove = useCallback((): void => {
    if (!storage) {
      return
    }

    storage.removeItem(storageKey)
  }, [storage, storageKey])

  // initialize React state lazily from storage if available, otherwise use initValue
  const [value, setValue] = useState<T | null>(() => get() ?? initValue ?? null)

  // keeps a ref in sync with the latest React state so that the setter can read the current value
  // without capturing it as a dependency (which would re-create the callback on every state change)
  const valueRef = useRef<T | null>(value)

  useEffect(() => {
    valueRef.current = value
  }, [value])

  /**
   * Effect for keeping state in sync with other browser tabs/windows (via the 'storage' event) and
   * other components in the same tab (via custom events).
   */
  useEffect(() => {
    if (
      typeof window === 'undefined' ||
      typeof window.addEventListener !== 'function' ||
      typeof window.removeEventListener !== 'function'
    ) {
      return
    }

    // handle `storage` events from other tabs/windows and update the value accordingly
    const onStorage = (e: StorageEvent) => {
      // ignore events from unrelated storage areas (should not happen, but just in case)
      if (e.storageArea !== storage) {
        return
      }

      switch (true) {
        // the whole storage was cleared
        case e.key === null:
          setValue(null)
          break

        // exactly our key was changed
        case e.key === storageKey:
          setValue(get())
          break
      }
    }

    // handle custom events from the same tab/window and update the value accordingly
    const onCustom = (e: Event) => {
      if (!(e instanceof CustomEvent)) {
        return
      }

      const { senderId } = (e as StorageNotifyEvent).detail
      if (senderId === senderIdRef.current) {
        return
      }

      setValue(get())
    }

    window.addEventListener('storage', onStorage)
    window.addEventListener(eventName, onCustom)

    return () => {
      window.removeEventListener('storage', onStorage)
      window.removeEventListener(eventName, onCustom)
    }
  }, [storage, storageKey, eventName, get])

  /** Hook-level setter that mirrors the React `setState` signature (accepts either a value or an updater function). */
  const setValueHook = useCallback(
    (newValue: SetStateAction<T | null>) => {
      // use valueRef instead of get() so that the updater function receives the current React state
      // rather than the value from storage (the two can differ when storage has never been written to)
      const next = typeof newValue === 'function' ? (newValue as (p: T | null) => T | null)(valueRef.current) : newValue

      if (next === null) {
        remove()
      } else {
        set(next)
      }

      setValue(next)

      if (typeof window === 'object' && typeof window.dispatchEvent === 'function') {
        window.dispatchEvent(new CustomEvent(eventName, { detail: { senderId: senderIdRef.current } }))
      }
    },
    [set, remove, eventName]
  )

  return [value, setValueHook]
}

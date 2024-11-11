import { useEffect, useState, type Dispatch, type SetStateAction } from 'react'

/**
 * The list of keys used in the storage.
 */
export enum UsedStorageKeys {
  UISettings = 'ui-settings',
  SessionsList = 'sessions-list',
  SessionsLastUsed = 'sessions-last-used',
  NewSessionStatusCode = 'ns-status-code',
  NewSessionHeadersList = 'ns-headers-list',
  NewSessionSessionDelay = 'ns-session-delay',
  NewSessionResponseBody = 'ns-response-body',
  NewSessionDestroyCurrentSession = 'ns-destroy-current',
  SessionDetailsShellTab = 'sd-selected-shell-tab',
  SessionDetailsCodeTab = 'sd-selected-code-tab',
  RequestDetailsHeadersExpand = 'rd-headers-expand',
}

export type StorageArea = 'local' | 'session'

/**
 * Hook to get and set a value in the storage. The value is stored as JSON. The key is automatically prefixed with
 * `webhook-tester-v2-`.
 *
 * The 3rd element in the returned tuple is a function to remove the value from the storage (this will not trigger any
 * change in the state).
 */
export function useStorage<T>(
  initValue: T,
  key: UsedStorageKeys,
  area: StorageArea = 'session'
): readonly [T, Dispatch<SetStateAction<T>>, () => void] {
  const storage: Storage = area === 'local' ? localStorage : sessionStorage
  const storageKey = `webhook-tester-v2-${key}`
  const loaded: string | null = storage.getItem(storageKey)
  const [value, setValue] = useState<T>(loaded !== null ? JSON.parse(loaded) : initValue)

  // update the value in the storage when it changes
  useEffect(() => {
    storage.setItem(storageKey, JSON.stringify(value))
  }, [storage, storageKey, value])

  return [
    value,
    setValue,
    (): void => {
      storage.removeItem(storageKey)
    },
  ]
}

import { useLocalStorage, useSessionStorage } from '@mantine/hooks'

/**
 * Returns the storage key for the given postfix.
 */
export function storageKey(postfix: string): string {
  return `webhook-tester-v2-${postfix}`
}

/**
 * Hook to get and set the last used SID (Session ID) and functions to update it.
 */
export function useLastUsedSID(): readonly [string | undefined, (value: string | undefined | null) => void] {
  const [sID, setSID, removeSID] = useLocalStorage<string | undefined>({
    key: storageKey('last-used-sid'),
    defaultValue: undefined,
  })

  return [
    sID,
    (newValue: string | undefined | null): void => {
      if (newValue) {
        setSID(newValue)
      } else {
        removeSID()
      }
    },
  ]
}

/**
 * Hook to get and set the last used RID (Request ID) and functions to update it.
 */
export function useLastUsedRID(): readonly [string | undefined, (value: string | undefined | null) => void] {
  const [rID, setRID, removeRID] = useSessionStorage<string | undefined>({
    key: storageKey('last-used-rid'),
    defaultValue: undefined,
  })

  return [
    rID,
    (newValue: string | undefined | null): void => {
      if (newValue) {
        setRID(newValue)
      } else {
        removeRID()
      }
    },
  ]
}

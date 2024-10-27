import { useLocalStorage, useSessionStorage } from '@mantine/hooks'

const storageKeyPrefix = 'webhook-tester-v2-last-used'

export function useLastUsedSID(): readonly [string | undefined, (value: string | undefined | null) => void] {
  const [sID, setSID, removeSID] = useLocalStorage<string | undefined>({
    key: `${storageKeyPrefix}-sid`,
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

export function useLastUsedRID(): readonly [string | undefined, (value: string | undefined | null) => void] {
  const [rID, setRID, removeRID] = useSessionStorage<string | undefined>({
    key: `${storageKeyPrefix}-rid`,
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

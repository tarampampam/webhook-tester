import { useLocalStorage, useSessionStorage } from '@mantine/hooks'

const storageKeyPrefix = 'webhook-tester-v2-last-used'

export function useLaseUsedSID(): readonly [string | undefined, (value: string | undefined) => void] {
  const [sID, setSID, removeSID] = useLocalStorage<string | undefined>({
    key: `${storageKeyPrefix}-sid`,
    defaultValue: undefined,
  })

  return [
    sID,
    (newValue: string | undefined): void => {
      if (newValue) {
        setSID(newValue)
      } else {
        removeSID()
      }
    },
  ]
}

export function useLaseUsedRID(): readonly [string | undefined, (value: string | undefined) => void] {
  const [rID, setRID, removeRID] = useSessionStorage<string | undefined>({
    key: `${storageKeyPrefix}-rid`,
    defaultValue: undefined,
  })

  return [
    rID,
    (newValue: string | undefined): void => {
      if (newValue) {
        setRID(newValue)
      } else {
        removeRID()
      }
    },
  ]
}

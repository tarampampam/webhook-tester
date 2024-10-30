import React, { createContext, useContext } from 'react'
import { UsedStorageKeys, useStorage } from './use-storage'

export type LastUsedContext = {
  lastUsedSID: string | null
  setLastUsedSID: (newSID: string | null) => void
  lastUsedRID: string | null
  setLastUsedRID: (newRID: string | null) => void
}

const lastUsedContext = createContext<LastUsedContext>({
  lastUsedSID: null,
  setLastUsedSID: () => {
    throw new Error('The LastUsedProvider is not initialized')
  },
  lastUsedRID: null,
  setLastUsedRID: () => {
    throw new Error('The LastUsedProvider is not initialized')
  },
})

export const LastUsedProvider = ({ children }: { children: React.JSX.Element }) => {
  const [lastUsedSID, setLastUsedSID, rmLastUsedSID] = useStorage<string | null>(
    null,
    UsedStorageKeys.LastUsedSID,
    'local'
  )
  const [lastUsedRID, setLastUsedRID, rmLastUsedRID] = useStorage<string | null>(
    null,
    UsedStorageKeys.LastUserRID,
    'session'
  )

  return (
    <lastUsedContext.Provider
      value={{
        lastUsedSID: lastUsedSID,
        setLastUsedSID: (newSID) => {
          setLastUsedSID(newSID)

          if (newSID === null) {
            rmLastUsedSID()
          }
        },
        lastUsedRID: lastUsedRID,
        setLastUsedRID: (newRID) => {
          setLastUsedRID(newRID)

          if (newRID === null) {
            rmLastUsedRID()
          }
        },
      }}
    >
      {children}
    </lastUsedContext.Provider>
  )
}

export function useLastUsed(): LastUsedContext {
  const ctx = useContext(lastUsedContext)

  if (!ctx) {
    throw new Error('useLastUsed must be used within a LastUsedProvider')
  }

  return ctx
}

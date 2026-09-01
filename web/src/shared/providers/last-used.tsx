import React, { createContext, type Dispatch, type PropsWithChildren, type SetStateAction, useContext } from 'react'
import { useStorage } from '../hooks/use-storage'

interface Context {
  /** The last-used session ID, or null if none has been set yet. */
  readonly lastUsedSID: string | null
  /** Updates the last-used session ID. Pass null to clear it. */
  readonly setLastUsedSID: Dispatch<SetStateAction<string | null>>
}

const ctx = createContext<Context | null>(null)

/** Persists the last-used session ID to localStorage so it survives page reloads. */
export const LastUsedProvider = ({ children }: PropsWithChildren): React.JSX.Element => {
  const [lastUsedSID, setLastUsedSID] = useStorage<string | null>(null, 'last-used-sid', 'local')

  return <ctx.Provider value={{ lastUsedSID, setLastUsedSID }}>{children}</ctx.Provider>
}

/** Hook that returns the last-used session ID and its setter. Must be called inside a LastUsedProvider. */
export const useLastUsed = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useLastUsed must be used within a LastUsedProvider')
  }

  return context
}

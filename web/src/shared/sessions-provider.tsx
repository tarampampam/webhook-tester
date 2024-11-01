import React, { createContext, useContext } from 'react'
import { UsedStorageKeys, useStorage } from './use-storage'

type SessionsContext = {
  sessions: ReadonlyArray<string>
  lastUsed: string | null
  addSession: (sID: string) => void
  removeSession: (...sID: Array<string>) => void
  setLastUsed: (sID: string | null) => void
}

const sessionsContext = createContext<SessionsContext>({
  sessions: [],
  lastUsed: null,
  addSession: () => {
    throw new Error('The SessionsProvider is not initialized')
  },
  removeSession: () => {
    throw new Error('The SessionsProvider is not initialized')
  },
  setLastUsed: () => {
    throw new Error('The SessionsProvider is not initialized')
  },
})

export const SessionsProvider = ({ children }: { children: React.JSX.Element }) => {
  const [sessions, setSessions] = useStorage<string[]>([], UsedStorageKeys.SessionsList, 'local')
  const [lastUsed, setLastUsed, rmLastUsed] = useStorage<string | null>(null, UsedStorageKeys.SessionsLastUsed, 'local')

  return (
    <sessionsContext.Provider
      value={{
        sessions,
        lastUsed,
        addSession: (sID) => setSessions((prev) => (prev.includes(sID) ? prev : [...prev, sID])),
        removeSession: (...list) => {
          // if the last used session is in the list, set it to null
          if (lastUsed && list.includes(lastUsed)) {
            setLastUsed(null)
          }

          // remove the sessions from state
          setSessions((prev) => prev.filter((s) => !list.includes(s)))
        },
        setLastUsed: (sID) => {
          // prevent state change if the session is same as the last used
          if (sID === lastUsed) {
            return
          }

          // if the session ID is null, remove it from the storage and set to null
          if (sID === null) {
            setLastUsed(null)
            rmLastUsed()

            return
          }

          // only if provided session ID is in the list of sessions, update the state
          if (sessions.includes(sID)) {
            setLastUsed(sID)
          }
        },
      }}
    >
      {children}
    </sessionsContext.Provider>
  )
}

export function useSessions(): Readonly<SessionsContext> {
  const ctx = useContext(sessionsContext)

  if (!ctx) {
    throw new Error('useSessions must be used within a SessionsProvider')
  }

  return ctx
}

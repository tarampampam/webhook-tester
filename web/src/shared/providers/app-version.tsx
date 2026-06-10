import React, { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { type Client } from '~/api'
import { anyToError } from '../utils/errors'

interface Context {
  /** The version string of the currently running binary, or null while loading or on fetch error. */
  readonly current: string | null
  /** The latest published version string, or null while loading or on fetch error. */
  readonly latest: string | null
  /** True if an update is available, false if up-to-date, null if either version is not yet known. */
  readonly updateAvailable: boolean | null
  /** The last fetch error from either version endpoint. Null if no error has occurred. */
  readonly error: Error | null
}

const ctx = createContext<Context | null>(null)

/**
 * Compares two semantic version strings (e.g., "1.2.3") and returns true if the latest version is newer
 * than the current version.
 */
const isNewerVersion = (current: string, latest: string): boolean => {
  const [cMaj = 0, cMin = 0, cPatch = 0] = current.split('.').map((s) => parseInt(s, 10) || 0)
  const [lMaj = 0, lMin = 0, lPatch = 0] = latest.split('.').map((s) => parseInt(s, 10) || 0)
  if (cMaj !== lMaj) {
    return lMaj > cMaj
  }
  if (cMin !== lMin) {
    return lMin > cMin
  }

  return lPatch > cPatch
}

/** Fetches the current and latest app versions on mount and exposes them to children via context. */
export const AppVersionProvider = ({
  api,
  children,
}: {
  api: Client
  children?: React.ReactNode
}): React.JSX.Element => {
  const [current, setCurrent] = useState<string | null>(null)
  const [latest, setLatest] = useState<string | null>(null)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    if (current) {
      return
    }

    const ctrl = new AbortController()

    api
      .currentVersion({ signal: ctrl.signal })
      .then((ver) => {
        setCurrent(ver)
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        setError(anyToError(err))
      })

    return () => ctrl.abort()
  }, [api, current])

  useEffect(() => {
    if (latest) {
      return
    }

    const ctrl = new AbortController()

    api
      .latestVersion({ signal: ctrl.signal })
      .then((ver) => {
        setLatest(ver)
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        setError(anyToError(err))
      })

    return () => ctrl.abort()
  }, [api, latest])

  const updateAvailable: boolean | null = useMemo(() => {
    if (current && latest) {
      return isNewerVersion(current, latest)
    }

    return null
  }, [current, latest])

  return <ctx.Provider value={{ current, latest, updateAvailable, error }}>{children}</ctx.Provider>
}

/**
 * A hook to access the current and latest app versions, as well as any error that occurred while fetching them.
 * Must be used within an AppVersionProvider.
 */
export const useAppVersion = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useAppVersion must be used within an AppVersionProvider')
  }

  return context
}

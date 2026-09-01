import React, { createContext, type PropsWithChildren, useContext, useEffect, useState } from 'react'
import { type Client } from '~/api'
import { anyToError } from '../utils/errors'

// Derived from the client return type - keeps the shape in sync with the OpenAPI schema automatically.
type AppConfig = Awaited<ReturnType<Client['getSettings']>>

interface Context {
  readonly config: AppConfig | null
  readonly error: Error | null
}

const ctx = createContext<Context | null>(null)

/**
 * Provider that fetches app config from the API once and exposes it to children via context.
 * Skips re-fetching once config is loaded; aborts the in-flight request on unmount.
 */
export const AppConfigProvider = ({ api, children }: PropsWithChildren<{ api: Client }>): React.JSX.Element => {
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    if (config) {
      return
    }

    const ctrl = new AbortController()

    api
      .getSettings({ signal: ctrl.signal })
      .then((cfg) => {
        setConfig(cfg)
        setError(null)
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        setError(anyToError(err))
      })

    return () => ctrl.abort()
  }, [api, config])

  return <ctx.Provider value={{ config, error }}>{children}</ctx.Provider>
}

/**
 * A hook to access the app config's context. Must be used only within the AppConfigProvider.
 */
export const useAppConfig = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useAppConfig must be used within an AppConfigProvider')
  }

  return context
}

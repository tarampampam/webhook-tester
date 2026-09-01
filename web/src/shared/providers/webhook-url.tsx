import React, { createContext, type PropsWithChildren, useContext, useMemo } from 'react'
import { useActiveSessionID } from '~/routing/hooks'
import { useAppConfig } from './app-config'

interface Context {
  readonly webhookURL: URL | null
}

const ctx = createContext<Context | null>(null)

/**
 * Provides the webhook URL for the current session. Pass sID and publicUrlRoot explicitly,
 * or use WebhookURLProvider.WithConfigAndRouting to pull them from the router and app config.
 */
export const WebhookURLProvider = Object.assign(
  ({
    sID,
    publicUrlRoot,
    children,
  }: PropsWithChildren<{
    sID: string | null
    publicUrlRoot: URL | null
  }>): React.JSX.Element => {
    const webhookURL = useMemo<URL | null>(() => {
      if (!sID) {
        return null
      }

      const baseURL: string | null = publicUrlRoot
        ? publicUrlRoot.toString()
        : typeof window !== 'undefined'
          ? window.location.origin
          : null

      if (!baseURL) {
        return null
      }

      // ensure baseURL ends with exactly one slash so sID resolves as a child path segment
      return Object.freeze(new URL(sID, baseURL.replace(/\/*$/, '/')))
    }, [sID, publicUrlRoot])

    return <ctx.Provider value={{ webhookURL }}>{children}</ctx.Provider>
  },
  {
    /**
     * Higher-order component that provides the webhook URL for the current session, pulling all necessary values
     * from other providers.
     */
    WithConfig: ({ children }: PropsWithChildren): React.JSX.Element => {
      const { config } = useAppConfig()
      const sID = useActiveSessionID()

      return (
        <WebhookURLProvider sID={sID} publicUrlRoot={config?.publicUrlRoot ?? null}>
          {children}
        </WebhookURLProvider>
      )
    },
  }
)

/** Returns the current webhook URL context. Throws if called outside WebhookURLProvider. */
export const useWebhookURL = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useWebhookURL must be used within a WebhookURLProvider')
  }

  return context
}

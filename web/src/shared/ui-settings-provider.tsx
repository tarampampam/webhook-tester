import React, { createContext, useContext, useEffect, useRef } from 'react'
import { UsedStorageKeys, useStorage } from './use-storage'

export type UISettings = {
  showRequestDetails: boolean
  autoNavigateToNewRequest: boolean
}

const defaults: Readonly<UISettings> = {
  showRequestDetails: true,
  autoNavigateToNewRequest: true,
}

export type UISettingsContext = {
  settings: Readonly<UISettings>
  ref: React.MutableRefObject<Readonly<UISettings>> // ref to the current settings object
  update(newSettings: Partial<Readonly<UISettings>>): void
}

const uiSettingsContext = createContext<UISettingsContext>({
  settings: defaults,
  ref: { current: defaults },
  update: () => {
    throw new Error('The UISettingsProvider is not initialized')
  },
})

export const UISettingsProvider = ({ children }: { children: React.JSX.Element }) => {
  const [settings, setSettings] = useStorage<UISettings>(defaults, UsedStorageKeys.UISettings, 'local')
  const ref = useRef<UISettings>(settings)

  useEffect(() => {
    ref.current = settings
  }, [settings])

  return (
    <uiSettingsContext.Provider
      value={{
        settings,
        ref,
        update: (newSettings) => setSettings((prev) => ({ ...prev, ...newSettings })),
      }}
    >
      {children}
    </uiSettingsContext.Provider>
  )
}

export function useUISettings(): UISettingsContext {
  const ctx = useContext(uiSettingsContext)

  if (!ctx) {
    throw new Error('useUISettings must be used within a UISettingsProvider')
  }

  return ctx
}

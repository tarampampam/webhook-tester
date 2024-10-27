import { useLocalStorage } from '@mantine/hooks'
import React, { createContext, useContext, useEffect, useRef } from 'react'
import { storageKey } from './use-storage'

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
  settingsRef: React.MutableRefObject<Readonly<UISettings>>
  updateSettings: (newSettings: Partial<Readonly<UISettings>>) => void
}

const uiSettingsContext = createContext<UISettingsContext>({
  settings: defaults,
  settingsRef: { current: defaults },
  updateSettings: () => {
    throw new Error('The UISettingsProvider is not initialized')
  },
})

export const UISettingsProvider = ({ children }: { children: React.JSX.Element }) => {
  const [settings, setSettings] = useLocalStorage<UISettings>({
    key: storageKey('ui-settings'),
    defaultValue: defaults,
  })

  const ref = useRef<UISettings>(settings)

  useEffect(() => {
    ref.current = settings
  }, [settings])

  return (
    <uiSettingsContext.Provider
      value={{
        settings: settings,
        settingsRef: ref,
        updateSettings: (newSettings) => setSettings((prev) => ({ ...prev, ...newSettings })),
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

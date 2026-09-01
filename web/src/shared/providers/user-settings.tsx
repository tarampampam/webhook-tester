import React, { createContext, type PropsWithChildren, useCallback, useContext } from 'react'
import { useStorage } from '../hooks/use-storage'

/**
 * The shape of the user settings data stored in the storage. It shouldn't be changed without a proper migration,
 * otherwise it may cause issues with the existing users' settings.
 *
 * If you need to add new settings, make sure to provide default values for them in the UserSettingsProvider
 * component, so that the existing users won't be affected by the change.
 */
type UserSettings = {
  readonly showRequestDetails: boolean
  readonly autoNavigateToNewRequest: boolean
  readonly showNativeRequestNotifications: boolean
}

const DEFAULT_USER_SETTINGS: Readonly<UserSettings> = {
  showRequestDetails: true,
  autoNavigateToNewRequest: true,
  showNativeRequestNotifications: false,
}

interface Context {
  readonly userSettings: Readonly<UserSettings>
  updateUserSettings(settings: Partial<UserSettings>): void
}

const ctx = createContext<Context | null>(null)

/**
 * Provider that persists user preferences to localStorage and exposes them to children via context.
 * Falls back to defaults when storage is unavailable.
 */
export const UserSettingsProvider = ({ children }: PropsWithChildren): React.JSX.Element => {
  const [settings, setSettings] = useStorage<UserSettings>(DEFAULT_USER_SETTINGS, 'user-settings', 'local')

  // merge partial updates into the current settings, falling back to defaults if storage is unavailable
  const updateUserSettings = useCallback(
    (patch: Partial<UserSettings>) => {
      setSettings((prev) => ({
        ...(prev ?? DEFAULT_USER_SETTINGS),
        ...patch,
      }))
    },
    [setSettings]
  )

  return (
    <ctx.Provider
      value={{
        userSettings: settings ?? DEFAULT_USER_SETTINGS,
        updateUserSettings,
      }}
    >
      {children}
    </ctx.Provider>
  )
}

/** A hook to access the user's settings context. Must be used only within the UserSettingsProvider. */
export const useUserSettings = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useUserSettings must be used within a UserSettingsProvider')
  }

  return context
}

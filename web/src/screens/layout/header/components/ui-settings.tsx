import type React from 'react'
import { Checkbox, Stack, Text } from '@mantine/core'
import { useBrowserNotifications, useUserSettings } from '~/shared'

export const UISettings = (): React.JSX.Element => {
  const {
    userSettings: { autoNavigateToNewRequest, showRequestDetails, showNativeRequestNotifications },
    updateUserSettings,
  } = useUserSettings()
  const { granted, request } = useBrowserNotifications()

  /** Handle the change of the native notifications setting */
  const handleNativeNotificationsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (granted) {
      updateUserSettings({ showNativeRequestNotifications: event.target.checked })
    } else {
      request().then((ok) => {
        updateUserSettings({ showNativeRequestNotifications: ok && event.target.checked })
      })
    }
  }

  return (
    <Stack>
      <Checkbox
        checked={autoNavigateToNewRequest}
        onChange={(event) => updateUserSettings({ autoNavigateToNewRequest: event.target.checked })}
        label="Automatically navigate to the new request"
      />
      <Checkbox
        checked={showRequestDetails}
        onChange={(event) => updateUserSettings({ showRequestDetails: event.target.checked })}
        label="Display request details"
      />
      <Checkbox
        checked={showNativeRequestNotifications}
        onChange={handleNativeNotificationsChange}
        label={
          <>
            <Text size="sm">Use native notifications for new requests (instead of the in-app ones)</Text>
            {!granted && (
              <Text size="sm" c="dimmed" fw={700}>
                Permission required
              </Text>
            )}
          </>
        }
      />
    </Stack>
  )
}

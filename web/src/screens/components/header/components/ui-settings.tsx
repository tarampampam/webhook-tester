import type React from 'react'
import { Checkbox, Stack, Text } from '@mantine/core'
import { useBrowserNotifications, useSettings } from '~/shared'

let count: number = 0

export const UISettings = (): React.JSX.Element => {
  console.debug(`🖌 UISettings rendering (${++count})`)

  const { autoNavigateToNewRequest, showRequestDetails, showNativeRequestNotifications, updateSettings } = useSettings()
  const { granted, request } = useBrowserNotifications()

  /** Handle the change of the native notifications setting */
  const handleNativeNotificationsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (granted) {
      updateSettings({ showNativeRequestNotifications: event.target.checked })
    } else {
      request().then((ok) => {
        updateSettings({ showNativeRequestNotifications: ok && !event.target.checked })
      })
    }
  }

  return (
    <Stack>
      <Checkbox
        checked={autoNavigateToNewRequest}
        onChange={(event) => updateSettings({ autoNavigateToNewRequest: event.target.checked })}
        label="Automatically navigate to the new request"
      />
      <Checkbox
        checked={showRequestDetails}
        onChange={(event) => updateSettings({ showRequestDetails: event.target.checked })}
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

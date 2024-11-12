import type React from 'react'
import { Button, Select, Stack } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { IconGrave2 } from '@tabler/icons-react'
import { notifications as notify } from '@mantine/notifications'
import { useData } from '~/shared'
import { pathTo, RouteIDs } from '~/routing'

export const SessionSwitch = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { allSessionIDs, session, destroySession } = useData()

  /** Switch to another session */
  const handleSwitchTo = (switchTo: string | null) => {
    if (switchTo) {
      navigate(pathTo(RouteIDs.Home))
    } else {
      throw new Error('No webhook ID to switch to')
    }
  }

  /** Destroy the current session */
  const handleDestroy = () => {
    if (session) {
      const thisSessionIdx: number | -1 = allSessionIDs.indexOf(session.sID)
      const switchTo: string | null = allSessionIDs[thisSessionIdx + 1] || allSessionIDs[thisSessionIdx - 1] || null

      destroySession(session.sID)
        .then(() => notify.show({ title: 'WebHook deleted', message: null, color: 'lime', autoClose: 3000 }))
        .then(() => {
          if (switchTo) {
            navigate(pathTo(RouteIDs.SessionAndRequest, switchTo))
          } else {
            navigate(pathTo(RouteIDs.Home))
          }
        })
        .catch(() => {
          notify.show({
            title: 'Failed to destroy the webhook',
            message: 'Please try again or reload the page',
            color: 'red',
            autoClose: 5000,
          })
        })
    } else {
      throw new Error('No active session')
    }
  }

  return (
    <Stack gap="xs" pb="0.25em">
      {!!session && (
        <Select
          label="Switch to a different webhook"
          placeholder="Select a webhook ID to switch to"
          comboboxProps={{ withinPortal: false }}
          checkIconPosition="right"
          data={allSessionIDs}
          value={session.sID}
          onChange={handleSwitchTo}
        />
      )}
      <Button
        variant="light"
        size="compact-sm"
        leftSection={<IconGrave2 size="1.1em" />}
        color="red"
        disabled={!session}
        onClick={handleDestroy}
      >
        Destroy this webhook
      </Button>
    </Stack>
  )
}

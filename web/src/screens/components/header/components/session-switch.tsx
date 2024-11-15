import React, { useCallback, useState } from 'react'
import { Button, Select, Stack } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { IconGrave2 } from '@tabler/icons-react'
import { notifications as notify } from '@mantine/notifications'
import { useData } from '~/shared'
import { pathTo, RouteIDs } from '~/routing'

export const SessionSwitch = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { allSessionIDs, session, destroySession } = useData()
  const [loading, setLoading] = useState<boolean>(false)

  /** Switch to another session */
  const handleSwitchTo = (switchTo: string | null) => {
    if (switchTo) {
      navigate(pathTo(RouteIDs.SessionAndRequest, switchTo))
    } else {
      throw new Error('No webhook ID to switch to')
    }
  }

  /** Destroy the current session */
  const handleDestroy = useCallback(() => {
    if (session) {
      const thisSessionIdx: number | -1 = allSessionIDs.indexOf(session.sID)
      const switchTo: string | null = allSessionIDs[thisSessionIdx + 1] || allSessionIDs[thisSessionIdx - 1] || null

      setLoading(true)

      destroySession(session.sID)
        .then(() => notify.show({ title: 'WebHook deleted', message: null, color: 'lime', autoClose: 3000 }))
        .then(() => {
          if (switchTo) {
            navigate(pathTo(RouteIDs.SessionAndRequest, switchTo))
          } else {
            navigate(pathTo(RouteIDs.Home))
          }
        })
        .then((slow) => slow)
        .catch((err) => {
          notify.show({
            title: 'Failed to destroy the webhook',
            message: String(err),
            color: 'red',
            autoClose: 5000,
          })
        })
        .finally(() => setLoading(false))
    } else {
      throw new Error('No active session')
    }
  }, [allSessionIDs, destroySession, navigate, session])

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
        disabled={!session || loading}
        onClick={handleDestroy}
      >
        Destroy this webhook
      </Button>
    </Stack>
  )
}

import { Button, Select, Stack } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconGrave2 } from '@tabler/icons-react'
import React, { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveSessionID } from '~/routing'
import { useSessions } from '~/shared'

const getNextOrPrevKey = <V,>(map: ReadonlyMap<string, V>, current: string): string | null => {
  const keys = [...map.keys()]
  const idx = keys.indexOf(current)

  if (idx === -1) {
    return null
  }

  return keys[idx + 1] ?? keys[idx - 1] ?? null
}

export const SessionSwitch = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { sessions, delSession } = useSessions()
  const sID = useActiveSessionID()
  const [loading, setLoading] = useState<boolean>(false)

  /** Switch to another session */
  const handleSwitchTo = (switchTo: string | null) => {
    if (switchTo) {
      navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID: switchTo }))
    } else {
      throw new Error('No webhook ID to switch to')
    }
  }

  /** Destroy the current session */
  const handleDestroy = useCallback(() => {
    if (!sID) {
      throw new Error('No active session')
    }

    setLoading(true)

    delSession(sID)
      .then(() => notify.show({ title: 'WebHook deleted', message: null, color: 'lime', autoClose: 3000 }))
      .then(() => {
        const switchTo = getNextOrPrevKey(sessions, sID)
        if (switchTo) {
          navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID: switchTo }))
        } else {
          navigate(pathTo(ROUTE_ID.Home))
        }
      })
      .catch((err) => {
        notify.show({
          title: 'Failed to destroy the webhook',
          message: String(err),
          color: 'red',
          autoClose: 5000,
        })
      })
      .finally(() => setLoading(false))
  }, [delSession, navigate, sID, sessions])

  const allSessionIDs = useMemo<Array<string>>(() => [...sessions.keys()], [sessions])

  return (
    <Stack gap="xs" pb="0.25em">
      {!!sessions.size && (
        <Select
          label="Switch to a different webhook"
          placeholder="Select a webhook ID to switch to"
          comboboxProps={{ withinPortal: false }}
          checkIconPosition="right"
          data={allSessionIDs}
          value={sID}
          onChange={handleSwitchTo}
        />
      )}
      <Button
        variant="light"
        size="compact-sm"
        leftSection={<IconGrave2 size="1.1em" />}
        color="red"
        disabled={!sessions.size || loading}
        onClick={handleDestroy}
      >
        Destroy this webhook
      </Button>
    </Stack>
  )
}

import type React from 'react'
import { Title } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import type { Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { useLastUsed } from '~/shared'

export default function HomeScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [navigate, { hash }] = [useNavigate(), useLocation()]
  const { lastUsedSID: lastSID, lastUsedRID: lastRID } = useLastUsed()

  if (hash) {
    // v1 used url hash (anchor) to store the current state (sID and rID). To improve the user experience, we should
    // redirect to the new URL if the hash appears in the following format:
    //  #/:sID/:rID

    const [sID, rID]: Array<string | undefined> = hash
      .replace(/^#\/+/, '')
      .split('/')
      .map((v) => v || undefined)
      .filter((v) => v && v.length === 36) // 36 characters is the length of a UUID

    if (sID && rID) {
      return <Navigate to={pathTo(RouteIDs.SessionAndRequest, sID, rID)} />
    } else if (sID) {
      return <Navigate to={pathTo(RouteIDs.SessionAndRequest, sID)} />
    }
  }

  // automatically redirect to the last used session and/or request, if available
  if (lastSID && lastRID) {
    return <Navigate to={pathTo(RouteIDs.SessionAndRequest, lastSID, lastRID)} />
  } else if (lastSID) {
    return <Navigate to={pathTo(RouteIDs.SessionAndRequest, lastSID)} />
  }

  // if no session is available, create a new one and redirect to it
  if (!lastSID) {
    const id = notify.show({
      title: 'Creating new session',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    apiClient
      .newSession({})
      .then((sInfo) => {
        notify.update({
          id,
          title: 'A new session has been created',
          message: `Session ID: ${sInfo.uuid}`,
          color: 'green',
          autoClose: 5000,
          loading: false,
        })

        navigate(pathTo(RouteIDs.SessionAndRequest, sInfo.uuid))
      })
      .catch(() => {
        notify.update({
          id,
          title: 'Failed to create session',
          message: 'Please try again later',
          color: 'red',
          loading: false,
        })
      })
  }

  return (
    <>
      <Title order={3} style={{ fontWeight: 300 }}>
        WebHook Tester allows you to easily test webhooks and other types of HTTP requests
      </Title>
      <Title order={5} c="dimmed" pt={5} style={{ fontWeight: 300 }}>
        Any requests sent to that URL are logged here instantly — you don&apos;t even have to refresh!
      </Title>
    </>
  )
}

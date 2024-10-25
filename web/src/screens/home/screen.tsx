import type React from 'react'
import { Title } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { Navigate, useNavigate } from 'react-router-dom'
import { useLastUsedRID, useLastUsedSID } from '~/shared'
import type { Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'

export default function HomeScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const navigate = useNavigate()
  const [lastSID, lastRID] = [useLastUsedSID()[0], useLastUsedRID()[0]]

  // automatically redirect to the last used session and/or request, if available
  if (lastSID && lastRID) {
    return <Navigate to={pathTo(RouteIDs.Session, lastSID, lastRID)} />
  } else if (lastSID) {
    return <Navigate to={pathTo(RouteIDs.Session, lastSID)} />
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
        notify.hide(id)

        navigate(pathTo(RouteIDs.Session, sInfo.uuid))
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

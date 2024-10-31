import { Title } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import React, { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import type { Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { useSessions } from '~/shared'

export default function HomeScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [navigate, { hash }] = [useNavigate(), useLocation()]
  const { addSession } = useSessions()
  const { sessions, lastUsed, setLastUsed } = useSessions()

  useEffect(() => {
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
        navigate(pathTo(RouteIDs.SessionAndRequest, sID, rID))

        return
      } else if (sID) {
        navigate(pathTo(RouteIDs.SessionAndRequest, sID))

        return
      }
    }

    // automatically redirect to the last used session, if available
    if (lastUsed) {
      notify.show({ title: 'Redirected to the last used WebHook', message: null })

      navigate(pathTo(RouteIDs.SessionAndRequest, lastUsed))

      return
    }

    // automatically redirect to the last created session, if available
    if (sessions.length) {
      notify.show({ title: 'Redirected to the last created WebHook', message: null })

      navigate(pathTo(RouteIDs.SessionAndRequest, sessions[sessions.length - 1]))

      return
    }

    // if no session is available, create a new one and redirect to it
    const id = notify.show({
      title: 'Creating new WebHook',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    apiClient
      .newSession({})
      .then((sInfo) => {
        notify.update({
          id,
          title: 'A new WebHook has been created',
          message: `Session ID: ${sInfo.uuid}`,
          color: 'green',
          autoClose: 5000,
          loading: false,
        })

        addSession(sInfo.uuid)
        setLastUsed(sInfo.uuid)

        navigate(pathTo(RouteIDs.SessionAndRequest, sInfo.uuid))
      })
      .catch(() => {
        notify.update({
          id,
          title: 'Failed to create WebHook',
          message: 'Please try again later',
          color: 'red',
          loading: false,
        })
      })
  }, [apiClient, addSession, hash, lastUsed, navigate, sessions, setLastUsed])

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

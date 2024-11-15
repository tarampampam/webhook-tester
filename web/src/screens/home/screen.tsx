import { Title } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import React, { useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import { useData } from '~/shared'

export function HomeScreen(): React.JSX.Element {
  const [navigate, { hash }] = [useNavigate(), useLocation()]
  const { lastUsedSID: last, allSessionIDs: all, newSession } = useData()

  // store the last used session ID and all session IDs in refs to prevent unnecessary re-renders
  const lastUsedSID = useRef<string | null>(last)
  const allSessionIDs = useRef<ReadonlyArray<string>>(all)

  // update the refs when the values change
  useEffect(() => { lastUsedSID.current = last }, [last]) // prettier-ignore
  useEffect(() => { allSessionIDs.current = all }, [all]) // prettier-ignore

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
  }, [hash, navigate])

  useEffect(() => {
    // automatically redirect to the last used session, if available
    if (lastUsedSID.current) {
      notify.show({ title: 'Redirected to the last used WebHook', message: null })

      navigate(pathTo(RouteIDs.SessionAndRequest, lastUsedSID.current))

      return
    }

    // automatically redirect to the last created session, if available
    if (allSessionIDs.current.length) {
      notify.show({ title: 'Redirected to the last created WebHook', message: null })

      navigate(pathTo(RouteIDs.SessionAndRequest, allSessionIDs.current[allSessionIDs.current.length - 1]))

      return
    }

    // if no session is available, create a new one and redirect to it
    const id = notify.show({
      title: 'Creating new WebHook',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    newSession({
      statusCode: 200,
      headers: { 'Content-Type': 'application/json' },
      responseBody: new TextEncoder().encode('"Hello, world!"'),
    })
      .then((sInfo) => {
        notify.update({
          id,
          title: 'A new WebHook has been created',
          message: `Session ID: ${sInfo.sID}`,
          color: 'green',
          autoClose: 5000,
          loading: false,
        })

        navigate(pathTo(RouteIDs.SessionAndRequest, sInfo.sID))
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
  }, [navigate, newSession])

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

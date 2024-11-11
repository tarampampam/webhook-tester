import { Title } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import React, { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import { useData } from '~/shared'

let count: number = 0

export function HomeScreen(): React.JSX.Element {
  console.debug(`🖌 HomeScreen rendering (${++count})`)

  const [navigate, { hash }] = [useNavigate(), useLocation()]
  const { switchToRequest, switchToSession, lastUsedSID, allSessionIDs, newSession } = useData()

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
        switchToRequest(sID, rID)
          .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, sID, rID)))
          .catch(() => {
            notify.show({ title: 'Failed to redirect to the request', message: null })

            switchToSession(sID)
              .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, sID)))
              .catch(() => {
                notify.show({ title: 'Failed to redirect to the WebHook', message: null, color: 'red' })

                navigate(pathTo(RouteIDs.Home))
              })
          })

        return
      } else if (sID) {
        switchToSession(sID)
          .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, sID)))
          .catch(() => {
            notify.show({ title: 'Failed to redirect to the WebHook', message: null, color: 'red' })

            navigate(pathTo(RouteIDs.Home))
          })

        return
      }
    }

    // automatically redirect to the last used session, if available
    if (lastUsedSID) {
      notify.show({ title: 'Redirected to the last used WebHook', message: null })

      switchToSession(lastUsedSID)
        .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, lastUsedSID)))
        .catch(() => {
          notify.show({ title: 'Failed to redirect to the last used WebHook', message: null, color: 'red' })
        })

      return
    }

    // automatically redirect to the last created session, if available
    if (allSessionIDs.length) {
      const lastCreated = allSessionIDs[allSessionIDs.length - 1]

      notify.show({ title: 'Redirected to the last created WebHook', message: null })

      switchToSession(lastCreated)
        .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, allSessionIDs[allSessionIDs.length - 1])))
        .catch(() => {
          notify.show({ title: 'Failed to redirect to the last created WebHook', message: null, color: 'red' })
        })

      return
    }

    // if no session is available, create a new one and redirect to it
    const id = notify.show({
      title: 'Creating new WebHook',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    newSession({})
      .then((sInfo) => {
        notify.update({
          id,
          title: 'A new WebHook has been created',
          message: `Session ID: ${sInfo.sID}`,
          color: 'green',
          autoClose: 5000,
          loading: false,
        })

        switchToSession(sInfo.sID)
          .then(() => navigate(pathTo(RouteIDs.SessionAndRequest, sInfo.sID)))
          .catch(() => notify.show({ title: 'Failed to redirect to the new WebHook', message: null, color: 'red' }))
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
  }, [allSessionIDs, hash, lastUsedSID, navigate, newSession, switchToRequest, switchToSession])

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

import { notifications as notify } from '@mantine/notifications'
import React, { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import type { Client } from '~/api'
import { sessionToUrl, useLastUsedSID } from '~/shared'
import { useLayoutOutletContext } from '../layout'
import { RequestsList, SessionDetails } from './components'

export default function SessionAndRequestScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [{ sID }, { rID }] = [
    useParams<Readonly<{ sID: string }>>() as { sID: string }, // I'm sure that sID is always present here
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this layout
  ]
  const setLastUsedSID = useLastUsedSID()[1]
  const [webHookUrl, setWebHookUrl] = useState<URL>(sessionToUrl(sID))
  const { setNavBar, emitWebHookUrlChange } = useLayoutOutletContext()
  const closeSub = useRef<(() => void) | null>(null)

  useEffect((): (() => void) => {
    setNavBar(<RequestsList sID={sID} rID={rID} />)
    setLastUsedSID(sID) // store current session ID as the last used one

    const newWebHookUrl = sessionToUrl(sID)

    setWebHookUrl(newWebHookUrl) // update the current webhook url
    emitWebHookUrlChange(newWebHookUrl) // tell the parent layout that we have a new URL

    // subscribe to the session requests via WebSocket
    apiClient
      .subscribeToSessionRequests(sID, {
        onUpdate: (request): void => {
          console.log('New request:', request) // TODO: for debugging purposes
        },
        onError: (error): void => {
          notify.show({ title: 'An error occurred with websocket', message: String(error), color: 'orange' })
        },
      })
      .then((closer): void => {
        closeSub.current = closer // save the closer function to call it when the component unmounts
      })
      .catch(console.error)

    // on unmount
    return (): void => {
      setNavBar(null) // clear the navbar

      if (closeSub.current) {
        closeSub.current() // close the subscription
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sID]) // do NOT add setLastUsedSID here to avoid infinite loop

  return <SessionDetails webHookUrl={webHookUrl} />
}

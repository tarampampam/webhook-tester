import { Blockquote } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconInfoCircle } from '@tabler/icons-react'
import React, { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { APIErrorCommon, APIErrorNotFound, type Client } from '~/api'
import { sessionToUrl, useLastUsedSID } from '~/shared'
import { pathTo, RouteIDs } from '~/routing'
import { useLayoutOutletContext } from '../layout'
import { RequestsList, SessionDetails, type SessionProps } from './components'

export default function SessionAndRequestScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this layout
  ]
  const navigate = useNavigate()
  const [loading, setLoading] = useState<boolean>(false)
  const [sessionProps, setSessionProps] = useState<SessionProps | null>(null)
  const setLastUsedSID = useLastUsedSID()[1]
  const [webHookUrl, setWebHookUrl] = useState<URL>(sessionToUrl(sID))
  const { setNavBar, emitWebHookUrlChange } = useLayoutOutletContext()
  const closeSub = useRef<(() => void) | null>(null)

  useEffect((): (() => void) => {
    console.log('Session ID:', sID) // TODO: for debugging purposes

    setLoading(true)

    // get the session details (thus validate the session ID)
    apiClient
      .getSession(sID)
      .then((session): void => {
        setNavBar(<RequestsList sID={sID} rID={rID} />)
        setLastUsedSID(sID) // store current session ID as the last used one
        setSessionProps(
          Object.freeze({
            statusCode: session.response.statusCode,
            headers: session.response.headers,
            delay: session.response.delay,
            body: session.response.body,
            createdAt: session.createdAt,
          })
        )

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
      })
      .catch((err) => {
        setNavBar(<></>) // set an empty navbar
        emitWebHookUrlChange(null)

        // if the session does not exist, show an error message and redirect to the home screen
        if (err instanceof APIErrorNotFound || err instanceof APIErrorCommon) {
          notify.show({
            title: 'Session not found',
            message: err instanceof APIErrorNotFound ? `The session with ID ${sID} does not exist` : String(err),
            color: 'red',
          })

          navigate(pathTo(RouteIDs.Home)) // redirect to the home screen

          return
        }

        console.error(err)
      })
      .finally(() => setLoading(false))

    // on unmount
    return (): void => {
      setNavBar(null) // clear the navbar

      if (closeSub.current) {
        closeSub.current() // close the subscription
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sID, apiClient]) // do NOT add setLastUsedSID here to avoid infinite loop

  return (
    <>
      <SessionDetails webHookUrl={webHookUrl} loading={loading} sessionProps={sessionProps} />

      <Blockquote my="lg" color="blue" icon={<IconInfoCircle />}>
        Click &quot;New URL&quot; (in the top right corner) to create a new url with the ability to customize status
        code, response body, etc.
      </Blockquote>
    </>
  )
}

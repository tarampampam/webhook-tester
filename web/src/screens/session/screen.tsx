import { Blockquote } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconInfoCircle } from '@tabler/icons-react'
import React, { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { APIErrorCommon, APIErrorNotFound, type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { sessionToUrl, useLastUsedSID, useLastUsedRID, useUISettings } from '~/shared'
import { useLayoutOutletContext } from '../layout'
import { RequestDetails, SessionDetails, type SessionProps } from './components'

export default function SessionAndRequestScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this layout
  ]
  const navigate = useNavigate()
  const [loading, setLoading] = useState<boolean>(false)
  const [sessionProps, setSessionProps] = useState<SessionProps | null>(null)
  const [setLastUsedSID, setLastUsedRID] = [useLastUsedSID()[1], useLastUsedRID()[1]]
  const uiSettings = useUISettings().settingsRef
  const { setListedRequests, setSID: setParentSID, setRID: setParentRID } = useLayoutOutletContext()
  const closeSub = useRef<(() => void) | null>(null)

  useEffect((): (() => void) => {
    setParentSID(sID) // set the parent layout's session ID
    setLoading(true)

    // 🚀 get the session details (thus validate the session ID)
    apiClient
      .getSession(sID)
      .then((session): void => {
        setSessionProps(
          Object.freeze({
            statusCode: session.response.statusCode,
            headers: session.response.headers,
            delay: session.response.delay,
            body: session.response.body,
            createdAt: session.createdAt,
          })
        )

        // 🚀 get the session requests
        apiClient // TODO: load session and requests in parallel
          .getSessionRequests(sID)
          .then((requests) => {
            // update the list of requests
            setListedRequests(
              requests.map((request) =>
                Object.freeze({
                  id: request.uuid,
                  method: request.method,
                  clientAddress: request.clientAddress,
                  capturedAt: request.capturedAt,
                })
              )
            )
          })
          .catch((err) => {
            notify.show({ title: 'An error occurred while fetching requests', message: String(err), color: 'red' })

            console.error(err)
          })

        setLastUsedSID(sID) // store current session ID as the last used one

        // 🚀 subscribe to the session requests via WebSocket
        apiClient
          .subscribeToSessionRequests(sID, {
            onUpdate: (request): void => {
              // append the new request in front of the list
              setListedRequests((prev) => [
                Object.freeze({
                  id: request.uuid,
                  method: request.method,
                  clientAddress: request.clientAddress,
                  capturedAt: request.capturedAt,
                }),
                ...prev,
              ])

              if (uiSettings.current.autoNavigateToNewRequest) {
                navigate(pathTo(RouteIDs.SessionAndRequest, sID, request.uuid)) // navigate to the new request
              }
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
        // if the session does not exist, show an error message and redirect to the home screen
        if (err instanceof APIErrorNotFound || err instanceof APIErrorCommon) {
          notify.show({
            title: 'Session not found',
            message: err instanceof APIErrorNotFound ? `The session with ID ${sID} does not exist` : String(err),
            color: 'red',
          })

          // clear the last used session ID and request ID
          setLastUsedSID(null)
          setLastUsedRID(null)

          navigate(pathTo(RouteIDs.Home)) // redirect to the home screen

          return
        }

        console.error(err)
      })
      .finally(() => setLoading(false))

    // on unmount
    return (): void => {
      if (closeSub.current) {
        closeSub.current() // close the subscription
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sID, apiClient]) // do NOT add setLastUsedSID here to avoid infinite loop

  useEffect((): void => {
    if (rID) {
      setParentRID(rID)
      setLastUsedRID(rID)
    } else {
      setParentRID(null)
      setLastUsedRID(null)
    }
  }, [rID]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    (!!rID && <RequestDetails apiClient={apiClient} sID={sID} rID={rID} />) || (
      <>
        <SessionDetails webHookUrl={sessionToUrl(sID)} loading={loading} sessionProps={sessionProps} />

        <Blockquote my="lg" color="blue" icon={<IconInfoCircle />}>
          Click &quot;New URL&quot; (in the top right corner) to create a new url with the ability to customize status
          code, response body, etc.
        </Blockquote>
      </>
    )
  )
}

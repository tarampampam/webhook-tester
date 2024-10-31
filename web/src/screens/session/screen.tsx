import { Blockquote } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconInfoCircle } from '@tabler/icons-react'
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { APIErrorCommon, APIErrorNotFound, type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { sessionToUrl, useSessions, useUISettings } from '~/shared'
import { useLayoutOutletContext } from '../layout'
import { RequestDetails, SessionDetails, type SessionProps } from './components'

export default function SessionAndRequestScreen({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here because it's required in the route
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this layout
  ]
  const navigate = useNavigate()
  const { sessions, setLastUsed, removeSession } = useSessions()
  const [loading, setLoading] = useState<boolean>(false)
  const [sessionProps, setSessionProps] = useState<SessionProps | null>(null)
  const { ref: uiSettings } = useUISettings()
  const { setListedRequests, setSID: setParentSID, setRID: setParentRID, appSettings } = useLayoutOutletContext()
  const closeSub = useRef<(() => void) | null>(null)
  const appSettingsRef = useRef(appSettings)

  // store the app settings in the ref to avoid unnecessary re-renders
  useEffect(() => {
    appSettingsRef.current = appSettings
  }, [appSettings])

  /** Subscribe to the session requests via WebSocket */
  const subscribe = useCallback(
    (sID: string) => {
      if (closeSub.current) {
        closeSub.current() // close the previous subscription
      }

      closeSub.current = null // reset the closer function

      apiClient
        .subscribeToSessionRequests(sID, {
          onUpdate: (request): void => {
            // append the new request in front of the list
            setListedRequests((prev) => {
              let newList = [
                Object.freeze({
                  id: request.uuid,
                  method: request.method,
                  clientAddress: request.clientAddress,
                  capturedAt: request.capturedAt,
                }),
                ...prev,
              ]

              if (
                !!appSettingsRef.current &&
                appSettingsRef.current.setMaxRequestsPerSession &&
                newList.length > appSettingsRef.current.setMaxRequestsPerSession
              ) {
                newList = newList.slice(0, appSettingsRef.current.setMaxRequestsPerSession)
              }

              return newList
            })

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
    },
    [apiClient, navigate, setListedRequests, uiSettings]
  )

  /** Unsubscribe from the session requests */
  const unsubscribe = useCallback((): void => {
    if (closeSub.current) {
      closeSub.current() // close the subscription
    }
  }, [])

  /** Load the session details and the list of requests */
  const loadSessionAndRequests = useCallback(
    (sID: string): void => {
      setLoading(true)

      Promise.all([
        apiClient.getSession(sID), // 🚀 get the session details
        apiClient.getSessionRequests(sID), // 🚀 get the session requests
      ])
        .then(([session, requests]) => {
          setParentSID(session.uuid) // notify the parent layout about the session ID change
          setLastUsed(session.uuid) // store current session ID as the last used one
          subscribe(session.uuid) // subscribe to the session requests

          setSessionProps(
            Object.freeze({
              statusCode: session.response.statusCode,
              headers: session.response.headers,
              delay: session.response.delay,
              body: session.response.body,
              createdAt: session.createdAt,
            })
          )

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
        .catch((err: Error | unknown) => {
          // if the session does not exist, show an error message and redirect to the home screen
          if (err instanceof APIErrorNotFound || err instanceof APIErrorCommon) {
            setLastUsed(null) // reset the last used session ID
            removeSession(sID) // remove the session from the list

            notify.show({
              title: 'WebHook not found',
              message: err instanceof APIErrorNotFound ? `The WebHook with ID ${sID} does not exist` : String(err),
              color: 'orange',
            })
          } else {
            notify.show({ title: 'An error occurred', message: String(err), color: 'red' })

            console.error(err)
          }

          navigate(pathTo(RouteIDs.Home)) // redirect to the home screen
        })
        .finally(() => setLoading(false))
    },
    [apiClient, navigate, setLastUsed, removeSession, setListedRequests, setParentSID, subscribe]
  )

  /** Verify the session existence and remove the missing ones */
  const invalidateMissingSessions = useCallback(() => {
    Promise.allSettled(sessions.map((sID) => apiClient.getSession(sID))).then((results) => {
      const missingSessions = results
        .map((result, i) => (result.status === 'rejected' ? sessions[i] : null))
        .filter((sID): sID is string => sID !== null)

      if (missingSessions.length > 0) {
        removeSession(...missingSessions)
      }
    })
  }, [apiClient, removeSession, sessions])

  useEffect((): (() => void) => {
    // on mount
    loadSessionAndRequests(sID)

    if (sessions.length) {
      invalidateMissingSessions()
    }

    // on unmount
    return (): void => {
      setParentSID(null)

      unsubscribe()
    }
  }, [loadSessionAndRequests, sessions, invalidateMissingSessions, sID, setParentSID, unsubscribe])

  // notify the parent layout about the request ID change
  useEffect((): void => setParentRID(rID || null), [setParentRID, rID])

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

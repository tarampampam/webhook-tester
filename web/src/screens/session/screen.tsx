import { Blockquote } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconInfoCircle, IconRocket } from '@tabler/icons-react'
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import { type SessionEvents, useBrowserNotifications, useData, useSettings } from '~/shared'
import { RequestDetails, SessionDetails } from './components'

export function SessionAndRequestScreen(): React.JSX.Element {
  const navigate = useNavigate()
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here because it's required in the route
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this screen
  ]
  const [sessionLoading, setSessionLoading] = useState<boolean>(false)
  const [requestLoading, setRequestLoading] = useState<boolean>(false)
  const { session, request, switchToSession, switchToRequest, setRequestsCount } = useData()
  const {
    showNativeRequestNotifications: useNative,
    autoNavigateToNewRequest: autoNavigate,
    maxRequestsPerSession: maxRequests,
  } = useSettings()
  const { granted: bnGranted, show: bnShow } = useBrowserNotifications()

  // store some values in the ref to avoid unnecessary re-renders
  const bnGrantedRef = useRef<boolean>(bnGranted) // is native browser notifications granted?
  const useNativeRef = useRef<boolean>(useNative) // should use native browser notifications?
  const autoNavigateRef = useRef<boolean>(autoNavigate) // should auto-navigate to the new request?
  const stateSID = useRef<string | null>(session?.sID || null)
  const stateRID = useRef<string | null>(request?.rID || null)

  // auto-update the ref values
  useEffect(() => { bnGrantedRef.current = bnGranted }, [bnGranted]) // prettier-ignore
  useEffect(() => { useNativeRef.current = useNative }, [useNative]) // prettier-ignore
  useEffect(() => { autoNavigateRef.current = autoNavigate }, [autoNavigate]) // prettier-ignore
  useEffect(() => { stateSID.current = session?.sID || null }, [session]) // prettier-ignore
  useEffect(() => { stateRID.current = request?.rID || null }, [request]) // prettier-ignore

  /** The event listeners for the session */
  const listeners = useCallback(
    (): Partial<SessionEvents> => ({
      onNewRequest: (req): void => {
        // the in-app notification function to show the new request notification
        const showInAppNotification = (): void => {
          notify.show({
            title: 'New request received',
            message: `From ${req.clientAddress} with method ${req.method}`,
            icon: <IconRocket />,
            color: 'blue',
          })
        }

        // show a notification about the new request using the browser's native notification API,
        // if the permission is granted and the setting is enabled
        if (bnGrantedRef.current && useNativeRef.current) {
          bnShow('New request received', {
            body: `From ${req.clientAddress} with method ${req.method}`,
            autoClose: 5000,
          })
            // in case the notification is not shown, show the in-app notification
            .then((n) => {
              if (!n) {
                showInAppNotification()
              }
            })
            // do the same in case of an error
            .catch(showInAppNotification)
        } else {
          // otherwise, show the in-app notification
          showInAppNotification()
        }

        if (maxRequests && maxRequests > 0) {
          setRequestsCount(maxRequests)
        }

        // navigate to the new request if the setting is enabled
        if (autoNavigateRef.current) {
          navigate(pathTo(RouteIDs.SessionAndRequest, sID, req.rID)) // navigate to the new request
        }
      },
      // TODO: add another event handles
      onError: (err): void => {
        notify.show({
          title: 'An error occurred during the subscription to the new requests',
          message: String(err),
          color: 'red',
        })
      },
    }),
    [bnShow, navigate, sID, maxRequests, setRequestsCount]
  )

  /** The effect to switch to the session and request */
  useEffect(() => {
    Promise.allSettled([
      // if the session ID has changed, switch to the session
      stateSID.current !== sID
        ? (async (): Promise<void> => {
            try {
              // invoke the fast switching to the session (usually with the data from the database) and
              // get the slow operation
              const sessionSwitchSlow = await switchToSession(sID, listeners())

              setSessionLoading(true) // set the session loading state to true

              // start the slow operation (usually with the data from the server)
              await sessionSwitchSlow()
            } finally {
              setSessionLoading(false) // unset the session loading state
            }
          })()
        : Promise.resolve(),

      // if the request ID has changed, switch to the request
      stateRID.current !== rID
        ? (async (): Promise<void> => {
            try {
              // invoke the fast switching to the request (usually with the data from the database) and
              // get the slow operation
              const requestSwitchSlow = await switchToRequest(sID, rID ?? null)

              setRequestLoading(true) // set the request loading state to true

              // start the slow operation (usually with the data from the server)
              await requestSwitchSlow()
            } finally {
              setRequestLoading(false) // unset the request loading state
            }
          })()
        : Promise.resolve(),
    ] satisfies Array<Promise<void>>).then(([sessionSwitchResult, requestSwitchResult]) => {
      // if switching to the session failed
      if (sessionSwitchResult.status === 'rejected') {
        notify.show({
          title: 'Switching to the session failed',
          message: String(sessionSwitchResult.reason),
          color: 'red',
        })

        navigate(pathTo(RouteIDs.Home)) // navigate to the home screen

        return
      }

      // if switching to the request failed
      if (requestSwitchResult.status === 'rejected') {
        notify.show({
          title: 'Switching to the request failed',
          message: String(requestSwitchResult.reason),
          color: 'red',
        })

        navigate(pathTo(RouteIDs.SessionAndRequest, sID)) // navigate to the session screen

        return
      }
    })
  }, [sID, rID, listeners, navigate, switchToRequest, switchToSession])

  return (
    (!!request && <RequestDetails loading={requestLoading} />) || (
      <>
        <SessionDetails loading={sessionLoading} />
        <Blockquote my="lg" color="blue" icon={<IconInfoCircle />}>
          Click &quot;New URL&quot; (in the top right corner) to create a new url with the ability to customize status
          code, response body, etc.
        </Blockquote>
      </>
    )
  )
}

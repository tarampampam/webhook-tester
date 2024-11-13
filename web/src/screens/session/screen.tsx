import React, { useEffect, useRef } from 'react'
import { notifications as notify } from '@mantine/notifications'
import { Blockquote } from '@mantine/core'
import { IconInfoCircle, IconRocket } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { useBrowserNotifications, useData, useSettings } from '~/shared'
import { pathTo, RouteIDs } from '~/routing'
import { RequestDetails, SessionDetails } from './components'

export function SessionAndRequestScreen(): React.JSX.Element {
  const navigate = useNavigate()
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here because it's required in the route
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this screen
  ]
  const { request, switchToSession, switchToRequest } = useData()
  const { showNativeRequestNotifications: useNative, autoNavigateToNewRequest: autoNavigate } = useSettings()
  const { granted: bnGranted, show: bnShow } = useBrowserNotifications()

  // store some values in the ref to avoid unnecessary re-renders
  const bnGrantedRef = useRef<boolean>(bnGranted) // is native browser notifications granted?
  const useNativeRef = useRef<boolean>(useNative) // should use native browser notifications?
  const autoNavigateRef = useRef<boolean>(autoNavigate) // should auto-navigate to the new request?

  useEffect(() => {
    switchToSession(sID, {
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

        // navigate to the new request if the setting is enabled
        if (autoNavigateRef.current) {
          navigate(pathTo(RouteIDs.SessionAndRequest, sID, req.rID)) // navigate to the new request
        }
      },
    })
      .then(() =>
        switchToRequest(sID, rID ?? null).catch((err) =>
          notify.show({ title: 'Switching to request failed', message: String(err), color: 'red' })
        )
      )
      .catch((err) => notify.show({ title: 'Switching to session failed', message: String(err), color: 'red' }))
  }, [sID, rID, switchToSession, switchToRequest, bnShow, navigate])

  return (
    (!!request && <RequestDetails />) || (
      <>
        <SessionDetails />
        <Blockquote my="lg" color="blue" icon={<IconInfoCircle />}>
          Click &quot;New URL&quot; (in the top right corner) to create a new url with the ability to customize status
          code, response body, etc.
        </Blockquote>
      </>
    )
  )
}

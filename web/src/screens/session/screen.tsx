import { Blockquote } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { IconInfoCircle, IconRocket } from '@tabler/icons-react'
import dayjs from 'dayjs'
import React, { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveRequestID } from '~/routing'
import { useAppConfig, useBrowserNotifications, useRequests, useUserSettings } from '~/shared'
import { RequestDetails, SessionDetails } from './components'

export const SessionAndRequestScreen = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { sID } = useParams<{ sID: string }>() as Readonly<{ sID: string }>
  const [sessionLoading, setSessionLoading] = useState<boolean>(false)
  const { requests, setSessionID } = useRequests()
  const activeRequestID = useActiveRequestID()
  const request = (activeRequestID ? requests.get(activeRequestID) : null) ?? null
  const {
    userSettings: { showNativeRequestNotifications: useNative, autoNavigateToNewRequest: autoNavigate },
  } = useUserSettings()
  const { config } = useAppConfig()
  const { granted: bnGranted, show: bnShow } = useBrowserNotifications()

  // store in refs to avoid stale closures in effects
  const bnGrantedRef = useRef<boolean>(bnGranted)
  const useNativeRef = useRef<boolean>(useNative)
  const autoNavigateRef = useRef<boolean>(autoNavigate)
  // tracks request IDs that have already been seen to distinguish initial load from new WebSocket pushes
  const seenRequestIDsRef = useRef(new Set<string>())

  useEffect(() => { bnGrantedRef.current = bnGranted }, [bnGranted]) // prettier-ignore
  useEffect(() => { useNativeRef.current = useNative }, [useNative]) // prettier-ignore
  useEffect(() => { autoNavigateRef.current = autoNavigate }, [autoNavigate]) // prettier-ignore

  // suppress unused-var warning; config is accessed only for maxRequests side-effect via the provider
  void config

  useEffect(() => {
    seenRequestIDsRef.current = new Set()
    setSessionLoading(true)

    setSessionID(sID)
      .catch((err) => {
        notify.show({
          title: 'Switching to the session failed',
          message: String(err),
          color: 'red',
        })
        navigate(pathTo(ROUTE_ID.Home))
      })
      .finally(() => setSessionLoading(false))
  }, [sID, setSessionID, navigate])

  // detect new requests arriving via WebSocket and show notifications
  useEffect(() => {
    if (sessionLoading) {
      // while the initial load runs, mark every request as already seen so they don't trigger notifications
      for (const id of requests.keys()) {
        seenRequestIDsRef.current.add(id)
      }
      return
    }

    const newEntries = [...requests.entries()].filter(([id]) => !seenRequestIDsRef.current.has(id))

    for (const [id, req] of newEntries) {
      const showInAppNotification = (): void => {
        notify.show({
          title: 'New request received',
          message: `From ${req.clientAddress} with method ${req.method}`,
          icon: <IconRocket />,
          color: 'blue',
        })
      }

      if (bnGrantedRef.current && useNativeRef.current) {
        bnShow(`New request received (${dayjs(req.capturedAt).format('HH:mm:ss.SSS')})`, {
          body: `From ${req.clientAddress} with method ${req.method}`,
          tag: 'new-request',
          autoClose: 5000,
        })
          .then((n) => {
            if (!n) {
              showInAppNotification()
            }
          })
          .catch(showInAppNotification)
      } else {
        showInAppNotification()
      }

      if (autoNavigateRef.current) {
        navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID, rID: id }))
      }

      seenRequestIDsRef.current.add(id)
    }
  }, [requests, sessionLoading, sID, navigate, bnShow])

  return (
    (!!request && <RequestDetails loading={false} />) || (
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

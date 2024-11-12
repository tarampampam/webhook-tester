import React, { useEffect } from 'react'
import { notifications as notify } from '@mantine/notifications'
import { Blockquote } from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { useParams } from 'react-router-dom'
import { useData } from '~/shared'
import { RequestDetails, SessionDetails } from './components'

export function SessionAndRequestScreen(): React.JSX.Element {
  const { request, switchToSession, switchToRequest } = useData()
  const [{ sID }, { rID }] = [
    useParams<{ sID: string }>() as Readonly<{ sID: string }>, // I'm sure that sID is always present here because it's required in the route
    useParams<Readonly<{ rID?: string }>>(), // rID is optional for this layout
  ]

  useEffect(() => {
    switchToSession(sID)
      .then(() =>
        switchToRequest(sID, rID ?? null).catch((err) =>
          notify.show({ title: 'Switching to request failed', message: String(err), color: 'red' })
        )
      )
      .catch((err) => notify.show({ title: 'Switching to session failed', message: String(err), color: 'red' }))
  }, [sID, rID, switchToSession, switchToRequest])

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

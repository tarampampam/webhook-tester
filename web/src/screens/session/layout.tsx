import React, { useEffect } from 'react'
import { Outlet, useParams } from 'react-router-dom'
import { useLastUsedSID, useNavBar } from '~/shared'

export default function Layout(): React.JSX.Element {
  const { sID } = useParams<Readonly<{ sID: string }>>()
  const navBar = useNavBar()
  const setLastUsedSID = useLastUsedSID()[1]

  useEffect((): undefined | (() => void) => {
    if (sID) {
      navBar.setComponent(<>My navbar for {sID}</>)
      setLastUsedSID(sID)
    }

    return (): void => {
      navBar.setComponent(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sID, navBar.setComponent]) // do NOT add setLastUsedSID here to avoid infinite loop

  return (
    <div>
      <h1>Session Layout ({sID})</h1>

      <Outlet />
    </div>
  )
}

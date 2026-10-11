import React from 'react'
import { Navigate } from 'react-router-dom'
import { pathTo, ROUTE_ID } from '~/routing'
import { useLastUsed } from '~/shared'

export const HomeGuard = ({ children }: { children: React.ReactNode }): React.JSX.Element => {
  const { lastUsedSID } = useLastUsed()

  if (lastUsedSID) {
    return <Navigate to={pathTo(ROUTE_ID.Session, { sID: lastUsedSID })} replace />
  }

  return <>{children}</>
}

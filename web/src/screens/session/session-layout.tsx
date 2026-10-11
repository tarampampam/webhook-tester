import { Box } from '@mantine/core'
import React, { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { useActiveSessionID } from '~/routing'
import { useRequests } from '~/shared'

export const SessionLayout = (): React.JSX.Element => {
  const sID = useActiveSessionID()
  const { setSessionID, unsetSessionID } = useRequests()

  useEffect(() => {
    if (!sID) {
      return
    }

    const ctrl = new AbortController()
    void setSessionID(sID, { signal: ctrl.signal })

    return () => {
      ctrl.abort()
      unsetSessionID()
    }
  }, [sID, setSessionID, unsetSessionID])

  return (
    <Box m="md">
      <Outlet />
    </Box>
  )
}

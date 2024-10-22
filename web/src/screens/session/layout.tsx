import type React from 'react'
import { Outlet, useParams } from 'react-router-dom'

export default function Layout(): React.JSX.Element {
  const { sID } = useParams<Readonly<{ sID: string }>>()

  return (
    <div>
      <h1>Session Layout ({sID})</h1>

      <Outlet />
    </div>
  )
}

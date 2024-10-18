import type React from 'react'
import { Outlet } from 'react-router-dom'

export default function Layout(): React.JSX.Element {
  return (
    <main>
      <h1>Session Layout</h1>

      <Outlet />
    </main>
  )
}

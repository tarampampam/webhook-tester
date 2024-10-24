import React, { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { NavBarProvider } from '~/shared'
import { routes } from './routing'
import '@mantine/core/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/notifications/styles.css'
import '~/theme/app.css'

/** App component */
const App = (): React.JSX.Element => {
  return (
    <MantineProvider defaultColorScheme="auto">
      <Notifications />
      <NavBarProvider>
        <RouterProvider router={createBrowserRouter(routes)} />
      </NavBarProvider>
    </MantineProvider>
  )
}

const root = document.getElementById('root') as HTMLElement

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>
)

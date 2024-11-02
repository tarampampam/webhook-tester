import React, { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { routes } from './routing'
import { BrowserNotificationsProvider, SessionsProvider, UISettingsProvider } from './shared'
import '@mantine/core/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/notifications/styles.css'
import '~/theme/app.css'

dayjs.extend(relativeTime) // https://day.js.org/docs/en/plugin/relative-time

/** App component */
const App = (): React.JSX.Element => {
  return (
    <MantineProvider defaultColorScheme="auto">
      <Notifications />
      <BrowserNotificationsProvider>
        <UISettingsProvider>
          <SessionsProvider>
            <RouterProvider router={createBrowserRouter(routes)} />
          </SessionsProvider>
        </UISettingsProvider>
      </BrowserNotificationsProvider>
    </MantineProvider>
  )
}

const root = document.getElementById('root') as HTMLElement

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>
)

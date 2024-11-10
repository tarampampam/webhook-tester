import React, { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { Client } from './api'
import { Database } from './db'
import { createRoutes } from './routing'
import '@mantine/core/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/notifications/styles.css'
import '~/theme/app.css'
import { BrowserNotificationsProvider, DataProvider, SessionsProvider, UISettingsProvider } from './shared'

dayjs.extend(relativeTime) // https://day.js.org/docs/en/plugin/relative-time

/** App component */
const App = (): React.JSX.Element => {
  const api = new Client()
  const db = new Database()

  return (
    <MantineProvider defaultColorScheme="auto">
      <Notifications />
      <BrowserNotificationsProvider>
        <DataProvider api={api} db={db}>
          <UISettingsProvider>
            <SessionsProvider>
              <RouterProvider router={createBrowserRouter(createRoutes(api))} />
            </SessionsProvider>
          </UISettingsProvider>
        </DataProvider>
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

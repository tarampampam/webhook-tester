import React from 'react'
import { CodeHighlightAdapterProvider, createHighlightJsAdapter } from '@mantine/code-highlight'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import hljs from 'highlight.js/lib/core'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { Client } from '~/api'
import { Database } from '~/db'
import { createRoutes } from '~/routing'
import { appTheme, initializeHighlightJs } from '~/theme'
import '~/theme/highlight.css'
import '@mantine/core/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/notifications/styles.css'
import {
  AppConfigProvider,
  AppVersionProvider,
  BrowserNotificationsProvider,
  L10nProvider,
  LastUsedProvider,
  RequestsProvider,
  SessionsProvider,
  UserSettingsProvider,
} from './shared'
import '~/theme/app.css'

dayjs.extend(relativeTime) // https://day.js.org/docs/en/plugin/relative-time
initializeHighlightJs(hljs) // Initialize highlight.js with languages

const highlightJsAdapter = createHighlightJsAdapter(hljs)

/** App component */
const App = (): React.JSX.Element => {
  const api = new Client()
  const db = new Database()

  return (
    <MantineProvider theme={appTheme} defaultColorScheme="auto">
      <CodeHighlightAdapterProvider adapter={highlightJsAdapter}>
        <Notifications />
        <AppProviders api={api} db={db} errHandler={console.error}>
          <RouterProvider router={createBrowserRouter(createRoutes())} />
        </AppProviders>
      </CodeHighlightAdapterProvider>
    </MantineProvider>
  )
}

/**
 * Providers component that wraps the app with all the necessary context providers.
 * The order of providers is important.
 */
const AppProviders = ({
  api,
  db,
  errHandler,
  children,
}: {
  api: Client
  db: Database
  errHandler?: (err: Error) => void
  children: React.ReactNode
}): React.JSX.Element => {
  return (
    <L10nProvider>
      <AppVersionProvider api={api}>
        <AppConfigProvider api={api}>
          <BrowserNotificationsProvider>
            <UserSettingsProvider>
              <SessionsProvider api={api} db={db} errHandler={errHandler}>
                <RequestsProvider.WithConfig api={api} db={db} errHandler={errHandler}>
                  <LastUsedProvider>{children}</LastUsedProvider>
                </RequestsProvider.WithConfig>
              </SessionsProvider>
            </UserSettingsProvider>
          </BrowserNotificationsProvider>
        </AppConfigProvider>
      </AppVersionProvider>
    </L10nProvider>
  )
}

createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)

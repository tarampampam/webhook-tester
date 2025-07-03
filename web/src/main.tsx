import type React from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { CodeHighlightAdapterProvider, createHighlightJsAdapter } from '@mantine/code-highlight'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import hljs from 'highlight.js/lib/core'
import { Client } from '~/api'
import { Database } from '~/db'
import { createRoutes } from '~/routing'
import { BrowserNotificationsProvider, DataProvider, SettingsProvider } from './shared'
import { initializeHighlightJs } from '~/theme'
import '~/theme/highlight.css'
import '@mantine/core/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/notifications/styles.css'
import '~/theme/app.css'

dayjs.extend(relativeTime) // https://day.js.org/docs/en/plugin/relative-time
initializeHighlightJs(hljs) // Initialize highlight.js with languages

const highlightJsAdapter = createHighlightJsAdapter(hljs)

/** App component */
const App = (): React.JSX.Element => {
  const api = new Client()
  const db = new Database()

  return (
    <MantineProvider defaultColorScheme="auto">
      <CodeHighlightAdapterProvider adapter={highlightJsAdapter}>
        <Notifications />
        <BrowserNotificationsProvider>
          <SettingsProvider>
            <DataProvider api={api} db={db} errHandler={console.error}>
              <RouterProvider router={createBrowserRouter(createRoutes(api))} />
            </DataProvider>
          </SettingsProvider>
        </BrowserNotificationsProvider>
      </CodeHighlightAdapterProvider>
    </MantineProvider>
  )
}

createRoot(document.getElementById('root') as HTMLElement).render(<App />)

import React, { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { MantineProvider } from '@mantine/core'
import { NavBarProvider } from '~/shared'
import { routes } from './routing'
import '@mantine/core/styles.css'
import '~/theme/app.css'

/** App component */
const App = (): React.JSX.Element => {
  return (
    <MantineProvider
      // https://mantine.dev/theming/mantine-provider/
      defaultColorScheme="auto"
    >
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

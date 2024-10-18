import React, { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { routes } from './routing'
import '~/theme/app.scss'

/** App component */
const App = (): React.JSX.Element => {
  // render the app
  return <RouterProvider router={createBrowserRouter(routes)} />
}

// and here we go :D
createRoot(document.getElementById('root') as HTMLElement).render(
  <StrictMode>
    <App />
  </StrictMode>
)

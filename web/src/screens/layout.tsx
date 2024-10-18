import type React from 'react'
import { Outlet } from 'react-router-dom'
import { type Client } from '~/api'

const Header = (): React.JSX.Element => {
  return <header />
}

const Main = (): React.JSX.Element => {
  return (
    <main>
      <Outlet />
    </main>
  )
}

const Footer = (): React.JSX.Element => {
  return <footer />
}

export default function Layout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  apiClient.currentVersion().then((version) => console.log(version))
  apiClient.latestVersion().then((version) => console.log(version))

  return (
    <>
      <h1>Main layout</h1>

      <Header />
      <Main />
      <Footer />
    </>
  )
}

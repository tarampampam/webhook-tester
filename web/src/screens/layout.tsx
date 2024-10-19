import type React from 'react'
import { Link, Outlet } from 'react-router-dom'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '../routing'

const Header = (): React.JSX.Element => {
  return <header>Main layout Header</header>
}

const Main = (): React.JSX.Element => {
  return (
    <main>
      Main layout main
      <Outlet />
    </main>
  )
}

const Footer = (): React.JSX.Element => {
  return (
    <footer>
      Main layout footer
      <p>
        <Link to={pathTo(RouteIDs.Home)}>Home</Link>
      </p>
      <p>
        <Link to={pathTo(RouteIDs.Session, 'sID')}>Session</Link>
      </p>
      <p>
        <Link to={pathTo(RouteIDs.SessionRequest, 'sID', 'rID')}>Request</Link>
      </p>
      <p>
        <Link to={'/foobar-404'}>404</Link>
      </p>
    </footer>
  )
}

export default function Layout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  apiClient.currentVersion().then((version) => console.log(version))
  apiClient.latestVersion().then((version) => console.log(version))

  return (
    <>
      <Header />
      <Main />
      <Footer />
    </>
  )
}

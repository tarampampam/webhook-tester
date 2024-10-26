import { AppShell, Center, Loader, Text } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import React, { useEffect, useState } from 'react'
import { Outlet, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '../routing'
import { Header, type NewSessionOptions } from './components'

type ContextType = Readonly<{
  navBar: React.JSX.Element | null
  setNavBar: (_: React.JSX.Element | null) => void
  emitWebHookUrlChange: (_: URL | null) => void
}>

export default function DefaultLayout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const params = useParams<{ sID?: string; rID?: string }>()
  const navigate = useNavigate()
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  // const [maxRequestsPerSession, setMaxRequestsPerSession] = useState<number>(0)
  const [maxRequestBodySize, setMaxRequestBodySize] = useState<number>(0)
  const [sessionTTLSec, setSessionTTLSec] = useState<number>(0)
  const [navBar, setNavBar] = useState<React.JSX.Element | null>(null)
  const [webHookUrl, setWebHookUrl] = useState<URL | null>(null)

  useEffect(() => {
    // load current and latest versions on mount
    apiClient.currentVersion().then(setCurrentVersion).catch(console.error)
    apiClient.latestVersion().then(setLatestVersion).catch(console.error)

    // and load the settings
    apiClient
      .getSettings()
      .then((settings) => {
        // setMaxRequestsPerSession(settings.limits.maxRequests)
        setMaxRequestBodySize(settings.limits.maxRequestBodySize)
        setSessionTTLSec(settings.limits.sessionTTL)
      })
      .catch(console.error)
  }, [apiClient])

  /** Handles creating a new session and optionally destroying the current one. */
  const handleNewSessionCreate = async (s: NewSessionOptions) => {
    const id = notify.show({
      title: 'Creating new session',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    let newSessionID: string

    // create a new session
    try {
      newSessionID = (
        await apiClient.newSession({
          statusCode: s.statusCode,
          headers: Object.fromEntries(s.headers.map((h) => [h.name, h.value])),
          delay: s.delay,
          responseBody: new TextEncoder().encode(s.responseBody),
        })
      ).uuid
    } catch (err) {
      notify.update({
        id,
        title: 'Failed to create new session',
        message: String(err),
        color: 'red',
        loading: false,
      })

      return
    }

    // destroy the current session, if needed
    try {
      if (s.destroyCurrentSession && params.sID) {
        await apiClient.deleteSession(params.sID)
      }
    } catch (err) {
      notify.show({
        title: 'Failed to delete current session',
        message: String(err),
        color: 'red',
        autoClose: 5000,
      })
    }

    notify.update({
      id,
      title: 'A new session has started!',
      message: undefined,
      color: 'green',
      autoClose: 7000,
      loading: false,
    })

    navigate(pathTo(RouteIDs.SessionAndRequest, newSessionID)) // navigate to the new session
  }

  return (
    <AppShell
      header={{ height: 70 }}
      navbar={{ width: 300, breakpoint: 'sm', collapsed: { mobile: !navBarIsOpened } }}
      padding="md"
    >
      <AppShell.Header>
        <Header
          currentVersion={currentVersion}
          latestVersion={latestVersion}
          maxRequestBodySize={maxRequestBodySize}
          sessionTTLSec={sessionTTLSec}
          webHookUrl={webHookUrl}
          isBurgerOpened={navBarIsOpened}
          onBurgerClick={navBarHandlers.toggle}
          onNewSessionCreate={handleNewSessionCreate}
        />
      </AppShell.Header>

      <AppShell.Navbar p="md" withBorder={false}>
        {navBar ? ( // navBar may be replaced by a child component (<Outlet />)
          navBar
        ) : (
          <Center pt="2em">
            <Loader color="dimmed" size="1em" mr={8} mb={3} />
            <Text c="dimmed">Waiting for first request</Text>
          </Center>
        )}
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet context={{ navBar, setNavBar, emitWebHookUrlChange: setWebHookUrl } satisfies ContextType} />
      </AppShell.Main>
    </AppShell>
  )
}

export function useLayoutOutletContext(): ContextType {
  return useOutletContext<ContextType>()
}

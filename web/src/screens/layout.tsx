import { AppShell, ScrollArea } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import React, { useEffect, useState } from 'react'
import { Outlet, useNavigate, useOutletContext } from 'react-router-dom'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { sessionToUrl } from '~/shared'
import { Header, SideBar, type NewSessionOptions, type ListedRequest } from './components'

type ContextType = Readonly<{
  setListedRequests: (list: Array<ListedRequest> | ((prev: Array<ListedRequest>) => Array<ListedRequest>)) => void
  sID: string | null
  setSID: (sID: string | null) => void
  rID: string | null
  setRID: (rID: string | null) => void
}>

export default function DefaultLayout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const navigate = useNavigate()
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  const [[sID, setSID], [rID, setRID]] = [useState<string | null>(null), useState<string | null>(null)]
  const [listedRequests, setListedRequests] = useState<Array<ListedRequest>>([])
  const [appSettings, setAppSettings] = useState<
    Readonly<{
      setMaxRequestsPerSession: number
      maxRequestBodySize: number
      sessionTTLSec: number
    } | null>
  >(null)

  useEffect(() => {
    // load current and latest versions on mount
    apiClient.currentVersion().then(setCurrentVersion).catch(console.error)
    apiClient.latestVersion().then(setLatestVersion).catch(console.error)

    // and load the settings
    apiClient
      .getSettings()
      .then((settings) => {
        setAppSettings(
          Object.freeze({
            setMaxRequestsPerSession: settings.limits.maxRequests,
            maxRequestBodySize: settings.limits.maxRequestBodySize,
            sessionTTLSec: settings.limits.sessionTTL,
          })
        )
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
      if (s.destroyCurrentSession && !!sID) {
        await apiClient.deleteSession(sID)
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
      <AppShell.Header style={{ zIndex: 103 }}>
        <Header
          currentVersion={currentVersion}
          latestVersion={latestVersion}
          appSettings={appSettings}
          webHookUrl={(sID && sessionToUrl(sID)) || null}
          isBurgerOpened={navBarIsOpened}
          onBurgerClick={navBarHandlers.toggle}
          onNewSessionCreate={handleNewSessionCreate}
        />
      </AppShell.Header>

      <AppShell.Navbar p="md" pr={0} style={{ zIndex: 102 }} withBorder={false}>
        <AppShell.Section grow component={ScrollArea} pr="md">
          <SideBar sID={sID} rID={rID} requests={listedRequests} />
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet context={{ setListedRequests, sID, setSID, rID, setRID } satisfies ContextType} />
      </AppShell.Main>
    </AppShell>
  )
}

export function useLayoutOutletContext(): ContextType {
  return useOutletContext<ContextType>()
}

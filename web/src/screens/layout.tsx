import { Affix, AppShell, Box, Button, ScrollArea, Transition } from '@mantine/core'
import { useDisclosure, useWindowScroll } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import { IconArrowUp } from '@tabler/icons-react'
import React, { useEffect, useState } from 'react'
import { Outlet, useNavigate, useOutletContext } from 'react-router-dom'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { sessionToUrl } from '~/shared'
import { Header, type ListedRequest, type NewSessionOptions, SideBar } from './components'

type ContextType = Readonly<{
  setListedRequests: (
    list: ReadonlyArray<ListedRequest> | ((prev: ReadonlyArray<ListedRequest>) => ReadonlyArray<ListedRequest>)
  ) => void
  sID: string | null
  setSID: (sID: string | null) => void
  rID: string | null
  setRID: (rID: string | null) => void
}>

export default function DefaultLayout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const navigate = useNavigate()
  const [scroll, scrollTo] = useWindowScroll()
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  const [[sID, setSID], [rID, setRID]] = [useState<string | null>(null), useState<string | null>(null)]
  const [listedRequests, setListedRequests] = useState<ReadonlyArray<ListedRequest>>([])
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
          headers: s.headers ? Object.fromEntries(s.headers.map((h) => [h.name, h.value])) : undefined,
          delay: s.delay ? s.delay : undefined,
          responseBody: s.responseBody ? new TextEncoder().encode(s.responseBody) : undefined,
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

  /** Handles deleting a request. */
  const handleDeleteRequest = (sID: string, rIDtoRemove: string): void => {
    const request = listedRequests.find((r) => r.id === rIDtoRemove)

    // if the request is not found, show an error message
    if (!request) {
      notify.show({ title: 'Failed to delete request', message: 'Request not found', color: 'red', autoClose: 5000 })

      return
    }

    const requestIdx = listedRequests.findIndex((r) => r.id === rIDtoRemove)
    const [nextRequest, prevRequest]: Partial<[ListedRequest, ListedRequest]> = [
      listedRequests[requestIdx + 1],
      listedRequests[requestIdx - 1],
    ]

    // remove the request from the list
    setListedRequests((prev) => prev.filter((r) => r.id !== rIDtoRemove))

    // delete the request from the server
    apiClient.deleteSessionRequest(sID, rIDtoRemove).catch((err) => {
      notify.show({ title: 'Failed to delete request', message: String(err), color: 'red', autoClose: 5000 })

      // restore the request to the list
      setListedRequests((prev) => [...prev, request])

      console.error(err)
    })

    if (rID === rIDtoRemove) {
      // if the request is currently opened, navigate to the next one
      if (nextRequest) {
        navigate(pathTo(RouteIDs.SessionAndRequest, sID, nextRequest.id))

        return
      } else if (prevRequest) {
        // if there is no next request, navigate to the previous one
        navigate(pathTo(RouteIDs.SessionAndRequest, sID, prevRequest.id))

        return
      }

      // if there is no next request, navigate to the session
      navigate(pathTo(RouteIDs.SessionAndRequest, sID))
    }
  }

  /** Handles deleting all requests. */
  const handleDeleteAllRequests = (sID: string): void => {
    const backup = [...listedRequests]

    // remove all requests from the list
    setListedRequests([])

    // delete all requests from the server
    apiClient.deleteAllSessionRequests(sID).catch((err) => {
      notify.show({ title: 'Failed to delete requests', message: String(err), color: 'red', autoClose: 5000 })

      setListedRequests(backup)

      console.error(err)
    })

    // navigate to the session
    navigate(pathTo(RouteIDs.SessionAndRequest, sID))
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
        <AppShell.Section grow component={ScrollArea} pr="md" scrollbarSize={6}>
          <SideBar
            sID={sID}
            rID={rID}
            requests={listedRequests}
            onRequestDelete={handleDeleteRequest}
            onAllRequestsDelete={handleDeleteAllRequests}
          />
          <Box
            h="100%"
            w="100%"
            pos="absolute"
            onClick={() => {
              // on click outside the requests list, navigate to the session
              if (sID) {
                setRID(null)
                navigate(pathTo(RouteIDs.SessionAndRequest, sID))
              }
            }}
          />
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet context={{ setListedRequests, sID, setSID, rID, setRID } satisfies ContextType} />
      </AppShell.Main>

      <Affix position={{ bottom: 20, right: 20 }}>
        <Transition transition="slide-up" mounted={scroll.y > 0}>
          {(transitionStyles) => (
            <Button
              color="teal"
              leftSection={<IconArrowUp size="1.2em" />}
              style={transitionStyles}
              onClick={() => scrollTo({ y: 0 })}
            >
              Scroll to top
            </Button>
          )}
        </Transition>
      </Affix>
    </AppShell>
  )
}

export function useLayoutOutletContext(): ContextType {
  return useOutletContext<ContextType>()
}

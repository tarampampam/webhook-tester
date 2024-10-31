import { Affix, AppShell, Button, ScrollArea, Transition } from '@mantine/core'
import { useDisclosure, useWindowScroll } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import { IconArrowUp } from '@tabler/icons-react'
import React, { useCallback, useEffect, useState } from 'react'
import { Outlet, useNavigate, useOutletContext } from 'react-router-dom'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { sessionToUrl, useSessions } from '~/shared'
import { Header, type ListedRequest, type NewSessionOptions, SideBar } from './components'

type ContextType = Readonly<{
  setListedRequests: (
    list: ReadonlyArray<ListedRequest> | ((prev: ReadonlyArray<ListedRequest>) => ReadonlyArray<ListedRequest>)
  ) => void
  sID: string | null
  setSID: (sID: string | null) => void
  rID: string | null
  setRID: (rID: string | null) => void
  appSettings: Readonly<AppSettings> | null
}>

type AppSettings = {
  setMaxRequestsPerSession: number
  maxRequestBodySize: number
  sessionTTLSec: number
}

const JumpToTop = ({
  scroll,
  scrollTo,
}: {
  scroll: ReturnType<typeof useWindowScroll>[0]
  scrollTo: ReturnType<typeof useWindowScroll>[1]
}): React.JSX.Element => (
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
)

export default function DefaultLayout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const navigate = useNavigate()
  const [scroll, scrollTo] = useWindowScroll()
  const { sessions, addSession, removeSession, setLastUsed } = useSessions()
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  const [[sID, setSID], [rID, setRID]] = [useState<string | null>(null), useState<string | null>(null)]
  const [listedRequests, setListedRequests] = useState<ReadonlyArray<ListedRequest>>([])
  const [appSettings, setAppSettings] = useState<Readonly<AppSettings | null>>(null)

  // load current and latest versions + settings on mount
  useEffect(() => {
    apiClient.currentVersion().then(setCurrentVersion).catch(console.error)
    apiClient.latestVersion().then(setLatestVersion).catch(console.error)

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

  /** Handles creating a new session and optionally destroying the current one */
  const handleNewSessionCreate = useCallback(
    async (s: NewSessionOptions) => {
      const id = notify.show({ title: 'Creating new WebHook', message: null, autoClose: false, loading: true })

      let newSID: string

      // create a new session
      try {
        newSID = (
          await apiClient.newSession({
            statusCode: s.statusCode,
            headers: s.headers ? Object.fromEntries(s.headers.map((h) => [h.name, h.value])) : undefined,
            delay: s.delay ? s.delay : undefined,
            responseBody: s.responseBody ? new TextEncoder().encode(s.responseBody) : undefined,
          })
        ).uuid

        addSession(newSID)
      } catch (err) {
        notify.update({
          id,
          title: 'Failed to create new WebHook',
          message: String(err),
          color: 'red',
          loading: false,
        })

        return
      }

      // remove the current session if needed (do it after creating a new session and in background)
      if (s.destroyCurrentSession && !!sID) {
        apiClient
          .deleteSession(sID)
          .then(() => removeSession(sID))
          .catch((err) => {
            notify.show({
              title: 'Failed to delete current WebHook',
              message: String(err),
              color: 'red',
              autoClose: 5000,
            })

            console.error(err)
          })
      }

      notify.update({
        id,
        title: 'A new WebHook has been created!',
        message: null,
        color: 'green',
        autoClose: 7000,
        loading: false,
      })

      navigate(pathTo(RouteIDs.SessionAndRequest, newSID)) // navigate to the new session
    },
    [addSession, apiClient, navigate, removeSession, sID]
  )

  /** Handles deleting a request */
  const handleDeleteRequest = useCallback(
    (sID: string, ridToRemove: string): void => {
      const request = listedRequests.find((r) => r.id === ridToRemove)

      // if the request is not found, show an error message
      if (!request) {
        notify.show({ title: 'Failed to delete request', message: 'Request not found', color: 'red', autoClose: 5000 })

        return
      }

      const requestIdx: number | -1 = listedRequests.findIndex((r) => r.id === ridToRemove)
      const [nextRequest, prevRequest]: [ListedRequest | undefined, ListedRequest | undefined] = [
        requestIdx !== -1 ? listedRequests[requestIdx + 1] : undefined,
        requestIdx !== -1 ? listedRequests[requestIdx - 1] : undefined,
      ]

      // remove the request from the list
      setListedRequests((prev) => prev.filter((r) => r.id !== ridToRemove))

      // delete the request from the server
      apiClient.deleteSessionRequest(sID, ridToRemove).catch((err) => {
        notify.show({ title: 'Failed to delete request', message: String(err), color: 'red', autoClose: 5000 })

        // restore the request to the list
        setListedRequests((prev) => [...prev, request])

        console.error(err)
      })

      if (rID === ridToRemove) {
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
    },
    [apiClient, listedRequests, navigate, rID]
  )

  /** Handles deleting all requests */
  const handleDeleteAllRequests = useCallback(
    (sID: string): void => {
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
    },
    [apiClient, listedRequests, navigate]
  )

  /** Handles switching to a different session */
  const handleSessionSwitch = useCallback(
    (sID: string): void => {
      setRID(null)
      setSID(sID)
      setLastUsed(sID)

      notify.show({ title: 'Switched to another WebHook', message: sID, color: 'lime', autoClose: 3000 })

      navigate(pathTo(RouteIDs.SessionAndRequest, sID))
    },
    [navigate, setRID, setSID, setLastUsed]
  )

  /** Handles destroying a session */
  const handleSessionDestroy = useCallback(
    (sID: string): void => {
      apiClient
        .deleteSession(sID)
        .then(() => {
          const thisSessionIdx: number | -1 = sessions.indexOf(sID)
          const switchTo = sessions[thisSessionIdx + 1] || sessions[thisSessionIdx - 1] || null

          removeSession(sID)

          notify.show({ title: 'WebHook deleted', message: null, color: 'lime', autoClose: 3000 })

          if (switchTo) {
            setSID(switchTo)
            setLastUsed(switchTo)

            navigate(pathTo(RouteIDs.SessionAndRequest, switchTo)) // navigate to the next or previous session
          } else {
            setSID(null)
            setLastUsed(null)

            // if there are no more sessions, navigate to the home screen
            navigate(pathTo(RouteIDs.Home))
          }
        })
        .catch((err) => {
          notify.show({ title: 'Failed to delete session', message: String(err), color: 'red', autoClose: 5000 })

          console.error(err)
        })
    },
    [apiClient, navigate, removeSession, sessions, setLastUsed, setSID]
  )

  /** Handles clicking on the navbar */
  const handleNavbarClick = useCallback(
    (e: React.MouseEvent) => {
      // prevent this event firing on children
      if (e.currentTarget !== e.target) {
        return
      }

      if (sID) {
        setRID(null) // unset the request ID

        navigate(pathTo(RouteIDs.SessionAndRequest, sID))
      }
    },
    [navigate, sID, setRID]
  )

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
          sID={sID}
          appSettings={appSettings}
          webHookUrl={(sID && sessionToUrl(sID)) || null}
          isBurgerOpened={navBarIsOpened}
          onBurgerClick={navBarHandlers.toggle}
          onNewSessionCreate={handleNewSessionCreate}
          onSessionSwitch={handleSessionSwitch}
          onSessionDestroy={handleSessionDestroy}
        />
      </AppShell.Header>

      <AppShell.Navbar p="md" pr={0} style={{ zIndex: 102 }} withBorder={false} onClick={handleNavbarClick}>
        <AppShell.Section component={ScrollArea} pr="md" scrollbarSize={6}>
          <SideBar
            sID={sID}
            rID={rID}
            requests={listedRequests}
            onRequestDelete={handleDeleteRequest}
            onAllRequestsDelete={handleDeleteAllRequests}
          />
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet context={{ setListedRequests, sID, setSID, rID, setRID, appSettings } satisfies ContextType} />
      </AppShell.Main>

      <JumpToTop scroll={scroll} scrollTo={scrollTo} />
    </AppShell>
  )
}

export function useLayoutOutletContext(): ContextType {
  return useOutletContext<ContextType>()
}

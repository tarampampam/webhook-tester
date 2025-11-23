import { Affix, AppShell, Button, ScrollArea, Transition } from '@mantine/core'
import { useDisclosure, useWindowScroll } from '@mantine/hooks'
import { IconArrowUp } from '@tabler/icons-react'
import React, { useCallback, useEffect, useState } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '~/routing'
import { useData, useSettings } from '~/shared'
import { Header, SideBar } from './components'

export default function DefaultLayout({ api }: { api: Client }): React.JSX.Element {
  const navigate = useNavigate()
  const [scroll, scrollTo] = useWindowScroll()
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  const { updateSettings } = useSettings()
  const { session } = useData()

  // load current and latest versions&settings on mount
  useEffect(() => {
    const errHandler: (err: Error | unknown) => void = console.error

    api.currentVersion().then(setCurrentVersion).catch(errHandler)
    api.latestVersion().then(setLatestVersion).catch(errHandler)
    api
      .getSettings()
      .then((s) =>
        updateSettings({
          maxRequestsPerSession: s.limits.maxRequests,
          maxRequestBodySize: s.limits.maxRequestBodySize,
          sessionTTLSec: s.limits.sessionTTL,
          tunnelEnabled: s.tunnel.enabled,
          tunnelUrl: s.tunnel.url,
          publicUrlRoot: s.publicUrlRoot,
        })
      )
      .catch(errHandler)
  }, [updateSettings, api])

  /** Handles clicking on the navbar */
  const handleNavbarClick = useCallback(
    (e: React.MouseEvent) => {
      // prevent this event firing on children
      if (e.currentTarget !== e.target) {
        return
      }

      if (session) {
        navigate(pathTo(RouteIDs.SessionAndRequest, session.sID))
      }
    },
    [navigate, session]
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
          isBurgerOpened={navBarIsOpened}
          onBurgerClick={navBarHandlers.toggle}
        />
      </AppShell.Header>

      <AppShell.Navbar p="md" pr={0} style={{ zIndex: 102 }} withBorder={false} onClick={handleNavbarClick}>
        <AppShell.Section component={ScrollArea} pr="md" scrollbarSize={6}>
          <SideBar />
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>

      <JumpToTop scroll={scroll} scrollTo={scrollTo} />
    </AppShell>
  )
}

const JumpToTop: React.FC<{
  scroll: ReturnType<typeof useWindowScroll>[0]
  scrollTo: ReturnType<typeof useWindowScroll>[1]
}> = ({ scroll, scrollTo }): React.JSX.Element => (
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

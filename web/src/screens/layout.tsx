import { AppShell, ScrollArea } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import React from 'react'
import { Outlet } from 'react-router-dom'
import { useActiveSessionID } from '~/routing'
import { useAppConfig, WebhookURLProvider } from '~/shared'
import { Header } from './layout/header'
import { SideBar } from './layout/sidebar'

export const DefaultLayout = (): React.JSX.Element => {
  const [navBarIsOpened, navBarHandlers] = useDisclosure()
  const { config } = useAppConfig()
  const sID = useActiveSessionID()

  return (
    <WebhookURLProvider sID={sID} publicUrlRoot={config?.publicUrlRoot ?? null}>
      <AppShell
        header={{ height: 70 }}
        navbar={{ width: 300, breakpoint: 'sm', collapsed: { mobile: !navBarIsOpened } }}
        padding="md"
      >
        <AppShell.Header style={{ zIndex: 103 }}>
          <Header isBurgerOpened={navBarIsOpened} onBurgerClick={navBarHandlers.toggle} />
        </AppShell.Header>

        <AppShell.Navbar p="md" pr={0} style={{ zIndex: 102 }} withBorder={false}>
          <AppShell.Section component={ScrollArea} pr="md" scrollbarSize={6}>
            <SideBar />
          </AppShell.Section>
        </AppShell.Navbar>

        <AppShell.Main>
          <Outlet />
        </AppShell.Main>
      </AppShell>
    </WebhookURLProvider>
  )
}

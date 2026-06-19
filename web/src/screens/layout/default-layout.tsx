import { AppShell, Drawer, useMantineTheme } from '@mantine/core'
import { useDisclosure, useMediaQuery } from '@mantine/hooks'
import React, { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { When } from '~/shared'
import { ThemeColor } from '~/theme'
import { Header } from './header/header'
import { Navbar } from './navbar/navbar'

const NAVBAR_WIDTH = 280

export const DefaultLayout = (): React.JSX.Element => {
  const [drawerOpened, { toggle, close }] = useDisclosure(false)
  const theme = useMantineTheme()
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`)

  useEffect(() => {
    if (!isMobile) {
      close()
    }
  }, [isMobile, close])

  return (
    <AppShell
      layout="alt"
      navbar={{ width: NAVBAR_WIDTH, breakpoint: 'sm', collapsed: { mobile: true } }}
      header={{ height: 60 }}
      styles={{
        main: {
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
        },
      }}
    >
      <AppShell.Navbar p="md" bg={ThemeColor.NavbarBg} withBorder={false}>
        <When
          condition={isMobile}
          wrapper={(children) => (
            <Drawer.Root opened={drawerOpened} onClose={close} position="left" size={NAVBAR_WIDTH}>
              <Drawer.Overlay blur={1} />
              <Drawer.Content bg={ThemeColor.NavbarBg}>
                <Drawer.Body p="md" style={{ height: '100%', overflow: 'hidden' }}>
                  {children}
                </Drawer.Body>
              </Drawer.Content>
            </Drawer.Root>
          )}
        >
          <Navbar />
        </When>
      </AppShell.Navbar>
      <AppShell.Header>
        <Header opened={drawerOpened} onToggle={toggle} />
      </AppShell.Header>
      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}

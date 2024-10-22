import { useDisclosure } from '@mantine/hooks'
import React, { useEffect, useState } from 'react'
import { Link, Outlet } from 'react-router-dom'
import { AppShell, Group, Burger, Image, Button } from '@mantine/core'
import {
  IconCopy,
  IconCirclePlusFilled,
  IconBrandGithubFilled,
  IconRefreshAlert,
  IconHelpHexagonFilled,
} from '@tabler/icons-react'
import type { SemVer } from 'semver'
import { type Client } from '~/api'
import { pathTo, RouteIDs } from '../routing'
import LogoTextSvg from '~/assets/togo-text.svg'

export default function Layout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [opened, { toggle }] = useDisclosure()
  const [currentVersion, setCurrentVersion] = useState<Readonly<SemVer> | null>(null)
  const [latestVersion, setLatestVersion] = useState<Readonly<SemVer> | null>(null)
  const isUpdateAvailable = currentVersion && latestVersion && currentVersion.compare(latestVersion) === -1

  useEffect(() => {
    apiClient
      .currentVersion()
      .then((v) => setCurrentVersion(v))
      .catch(console.error)

    apiClient
      .latestVersion()
      .then((v) => setLatestVersion(v))
      .catch(console.error)
  }, [apiClient])

  return (
    <AppShell
      header={{ height: 70 }}
      navbar={{ width: 300, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
          <Group>
            <Image
              src={LogoTextSvg}
              style={{ height: 20 }}
              title={currentVersion ? 'v' + currentVersion.toString() : undefined}
            />
            <Button.Group>
              <Button variant="default" size="xs" leftSection={<IconHelpHexagonFilled size={'1.3em'} />}>
                Help
              </Button>
              {isUpdateAvailable ? (
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<IconRefreshAlert size={'1.3em'} />}
                  component={Link}
                  to={__LATEST_RELEASE_LINK__}
                  target="_blank"
                >
                  Update available {latestVersion && <>(v{latestVersion.toString()})</>}
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<IconBrandGithubFilled size={'1.3em'} />}
                  component={Link}
                  to={__GITHUB_PROJECT_LINK__}
                  target="_blank"
                >
                  GitHub
                </Button>
              )}
            </Button.Group>
          </Group>
          <Group gap={5} visibleFrom="xs">
            <Button
              leftSection={<IconCopy size={'1.2em'} />}
              variant="gradient"
              gradient={{ from: 'teal', to: 'lime', deg: -90 }}
            >
              Copy Webhook URL
            </Button>
            <Button
              leftSection={<IconCirclePlusFilled size={'1.3em'} />}
              variant="gradient"
              gradient={{ from: 'blue', to: 'cyan', deg: 90 }}
            >
              New URL
            </Button>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md" withBorder={false}>
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
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}

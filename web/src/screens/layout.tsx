import { useDisclosure } from '@mantine/hooks'
import React, { useEffect, useState } from 'react'
import { Link, Outlet } from 'react-router-dom'
import { AppShell, Group, Center, Burger, Image, Button, Text, Loader } from '@mantine/core'
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
import { useNavBar } from '~/shared'
import LogoTextSvg from '~/assets/togo-text.svg'

export default function Layout({ apiClient }: { apiClient: Client }): React.JSX.Element {
  const [opened, { toggle }] = useDisclosure()
  const navBar = useNavBar()
  const [currentVersion, setCurrentVersion] = useState<SemVer | null>(null)
  const [latestVersion, setLatestVersion] = useState<SemVer | null>(null)
  const isUpdateAvailable = currentVersion && latestVersion && currentVersion.compare(latestVersion) === -1
  // const [webhookUrl, setWebhookUrl] = useState<URL>(
  //   new URL(`${window.location.origin}/33569c52-6c64-46eb-92b4-d3cb3824daa8`)
  // )

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
            <Button.Group visibleFrom="md">
              <buttons.OpenHelp />
              {isUpdateAvailable ? <buttons.OpenLatestRelease newVer={latestVersion} /> : <buttons.OpenGitHubProject />}
            </Button.Group>
          </Group>
          <Group visibleFrom="xs">
            <Button.Group>
              <buttons.CopyWebHookUrl />
              <buttons.CreateNewSession />
            </Button.Group>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md" withBorder={false}>
        <div>
          {navBar.component ? (
            navBar.component
          ) : (
            <Center pt="2em">
              <Loader color="dimmed" size="1em" mr={8} mb={3} /> <Text c="dimmed">Waiting for first request</Text>
            </Center>
          )}
        </div>
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>

      <AppShell.Aside>
        <p>
          <Link to={pathTo(RouteIDs.Home)}>Home</Link>
        </p>
        <p>
          <Link to={pathTo(RouteIDs.Session, 'sID')}>Session</Link>
        </p>
        <p>
          <Link to={pathTo(RouteIDs.Session, 'sID2')}>Session 2</Link>
        </p>
        <p>
          <Link to={pathTo(RouteIDs.SessionRequest, 'sID', 'rID')}>Request</Link>
        </p>
        <p>
          <Link to={pathTo(RouteIDs.SessionRequest, 'sID2', 'rID2')}>Request 2</Link>
        </p>
        <p>
          <Link to={'/foobar-404'}>404</Link>
        </p>
      </AppShell.Aside>
    </AppShell>
  )
}

const buttons = {
  OpenHelp: (): React.JSX.Element => (
    <Button variant="default" size="xs" leftSection={<IconHelpHexagonFilled size="1.3em" />}>
      Help
    </Button>
  ),
  OpenLatestRelease: ({ newVer }: { newVer?: SemVer }): React.JSX.Element => (
    <Button
      variant="default"
      size="xs"
      leftSection={<IconRefreshAlert size="1.3em" />}
      component={Link}
      to={__LATEST_RELEASE_LINK__}
      target="_blank"
    >
      Update available {newVer && <>(v{newVer.toString()})</>}
    </Button>
  ),
  OpenGitHubProject: (): React.JSX.Element => (
    <Button
      variant="default"
      size="xs"
      leftSection={<IconBrandGithubFilled size="1.3em" />}
      component={Link}
      to={__GITHUB_PROJECT_LINK__}
      target="_blank"
    >
      GitHub
    </Button>
  ),
  CopyWebHookUrl: (): React.JSX.Element => (
    <Button
      leftSection={<IconCopy size="1.2em" />}
      variant="gradient"
      gradient={{ from: 'teal', to: 'lime', deg: -90 }}
    >
      Copy Webhook URL
    </Button>
  ),
  CreateNewSession: (): React.JSX.Element => (
    <Button
      leftSection={<IconCirclePlusFilled size="1.3em" />}
      variant="gradient"
      gradient={{ from: 'blue', to: 'cyan', deg: 90 }}
    >
      New URL
    </Button>
  ),
}

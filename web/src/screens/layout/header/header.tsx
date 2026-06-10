import { Box, Burger, Button, Center, Group, Popover } from '@mantine/core'
import { useClipboard, useDisclosure } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import {
  IconAdjustmentsAlt,
  IconBrandGithubFilled,
  IconBuildingTunnel,
  IconCirclePlusFilled,
  IconCopy,
  IconHelpHexagonFilled,
  IconRefreshAlert,
  IconUsersGroup,
} from '@tabler/icons-react'
import React, { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { LogoText, useAppConfig, useAppVersion, useSessions, useWebhookURL } from '~/shared'
import { HelpModal, NewSessionModal, SessionSwitch, UISettings } from './components'

export const Header = ({
  isBurgerOpened = false,
  onBurgerClick = () => {},
}: {
  isBurgerOpened: boolean
  onBurgerClick: () => void
}): React.JSX.Element => {
  const clipboard = useClipboard({ timeout: 500 })
  const { current: currentVersion, latest: latestVersion, updateAvailable } = useAppVersion()
  const { webhookURL } = useWebhookURL()
  const { sessions } = useSessions()
  const { config } = useAppConfig()
  const [isNewSessionModalOpened, newSessionModalHandlers] = useDisclosure(false)
  const [isHelpModalOpened, helpModalHandlers] = useDisclosure(false)

  /** Handle copying the webhook URL to the clipboard */
  const handleCopyWebhookUrl = useCallback(() => {
    if (webhookURL) {
      clipboard.copy(webhookURL.toString())

      notify.show({
        title: 'Webhook URL copied',
        message: 'The URL has been copied to your clipboard.',
        color: 'lime',
        autoClose: 3000,
      })
    }
  }, [clipboard, webhookURL])

  return (
    <>
      <NewSessionModal opened={isNewSessionModalOpened} onClose={newSessionModalHandlers.close} />
      <HelpModal opened={isHelpModalOpened} onClose={helpModalHandlers.close} />

      <Group h="100%" px="md" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Burger opened={isBurgerOpened} onClick={onBurgerClick} hiddenFrom="sm" size="sm" />
        <Group>
          <Box display="contents" title={currentVersion ? `v${currentVersion}` : undefined}>
            <LogoText h={20} w="auto" />
          </Box>
          <Button.Group visibleFrom="sm">
            <Button
              variant="default"
              size="xs"
              leftSection={<IconHelpHexagonFilled size="1.3em" />}
              onClick={helpModalHandlers.open}
            >
              Help
            </Button>

            {config?.tunnel.enabled &&
              !!config.tunnel.url &&
              window.location.hostname !== config.tunnel.url.hostname && (
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<IconBuildingTunnel size="1.3em" />}
                  component={Link}
                  to={config.tunnel.url.toString()}
                  target="_blank"
                >
                  Tunnel
                </Button>
              )}
          </Button.Group>

          <Button.Group visibleFrom="lg">
            {updateAvailable && !!latestVersion ? (
              <Button
                variant="default"
                size="xs"
                leftSection={<IconRefreshAlert size="1.3em" />}
                component={Link}
                to={__LATEST_RELEASE_LINK__}
                rel="preload"
                target="_blank"
              >
                Update available {!!latestVersion && <>(v{latestVersion})</>}
              </Button>
            ) : (
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
            )}
          </Button.Group>
        </Group>
        <Group visibleFrom="xs">
          <Button.Group>
            <Popover width={250} position="bottom" shadow="md" withArrow>
              <Popover.Target>
                <Button
                  leftSection={<IconAdjustmentsAlt size="1.3em" />}
                  px="sm"
                  variant="gradient"
                  gradient={{ from: 'teal', to: 'lime', deg: 90 }}
                >
                  UI settings
                </Button>
              </Popover.Target>
              <Popover.Dropdown>
                <UISettings />
              </Popover.Dropdown>
            </Popover>

            {sessions.size > 1 && (
              <Popover position="bottom" shadow="md" withArrow>
                <Popover.Target>
                  <Button
                    px="sm"
                    variant="gradient"
                    gradient={{ from: 'lime', to: 'lime', deg: 90 }}
                    leftSection={<IconUsersGroup size="1.3em" />}
                  >
                    WebHooks
                  </Button>
                </Popover.Target>
                <Popover.Dropdown>
                  <Center>
                    <SessionSwitch />
                  </Center>
                </Popover.Dropdown>
              </Popover>
            )}

            <Button
              leftSection={<IconCopy size="1.2em" />}
              variant="gradient"
              gradient={{ from: 'lime', to: 'lime', deg: 90 }}
              color="lime"
              onClick={handleCopyWebhookUrl}
              disabled={!webhookURL}
              visibleFrom="md"
            >
              Copy Webhook URL
            </Button>

            <Button
              leftSection={<IconCirclePlusFilled size="1.3em" />}
              variant="gradient"
              gradient={{ from: 'lime', to: 'teal', deg: 90 }}
              onClick={newSessionModalHandlers.open}
            >
              New URL
            </Button>
          </Button.Group>
        </Group>
      </Group>
    </>
  )
}

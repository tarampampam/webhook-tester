import { Burger, Button, Center, Checkbox, Group, Image, Popover, Select, Stack, Text } from '@mantine/core'
import { useClipboard, useDisclosure } from '@mantine/hooks'
import { notifications as notify } from '@mantine/notifications'
import {
  IconAdjustmentsAlt,
  IconBrandGithubFilled,
  IconCirclePlusFilled,
  IconCopy,
  IconGrave2,
  IconHelpHexagonFilled,
  IconRefreshAlert,
  IconUsersGroup,
} from '@tabler/icons-react'
import React, { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import type { SemVer } from 'semver'
import LogoTextSvg from '~/assets/togo-text.svg'
import { useBrowserNotifications, useSessions, useUISettings } from '~/shared'
import HeaderHelpModal from './header-help-modal'
import HeaderNewSessionModal, { type NewSessionOptions } from './header-new-session-modal'

export default function Header({
  currentVersion,
  latestVersion,
  sID,
  appSettings = null,
  webHookUrl = null,
  isBurgerOpened = false,
  onBurgerClick = () => {},
  onNewSessionCreate = () => Promise.reject(),
  onSessionSwitch,
  onSessionDestroy,
}: {
  currentVersion: SemVer | null
  latestVersion: SemVer | null
  sID: string | null
  appSettings: {
    setMaxRequestsPerSession: number
    maxRequestBodySize: number
    sessionTTLSec: number
  } | null
  webHookUrl: URL | null
  isBurgerOpened: boolean
  onBurgerClick: () => void
  onNewSessionCreate?: (_: NewSessionOptions) => Promise<void>
  onSessionSwitch?: (to: string) => void
  onSessionDestroy?: (sID: string) => void
}): React.JSX.Element {
  const clipboard = useClipboard({ timeout: 500 })
  const { settings, update: updateSettings } = useUISettings()
  const { granted: browserNotificationsGranted, request: askForBrowserNotifications } = useBrowserNotifications()
  const { sessions } = useSessions()
  const [isUpdateAvailable, setIsUpdateAvailable] = useState<boolean>(false)
  const [isNewSessionModalOpened, newSessionModalHandlers] = useDisclosure(false)
  const [isHelpModalOpened, helpModalHandlers] = useDisclosure(false)
  const [isNewSessionLoading, setNewSessionLoading] = useState<boolean>(false)

  useEffect(() => {
    if (currentVersion && latestVersion) {
      setIsUpdateAvailable(currentVersion.compare(latestVersion) === -1)
    }
  }, [currentVersion, latestVersion])

  /** Handle copying the webhook URL to the clipboard */
  const handleCopyWebhookUrl = useCallback(() => {
    if (webHookUrl) {
      clipboard.copy(webHookUrl.toString())

      notify.show({
        title: 'Webhook URL copied',
        message: 'The URL has been copied to your clipboard.',
        color: 'lime',
        autoClose: 3000,
      })
    }
  }, [clipboard, webHookUrl])

  /** Handle creating a new session (interacting with the modal) */
  const handleNewSessionCreate = useCallback(
    async (options: NewSessionOptions) => {
      setNewSessionLoading(true)

      try {
        await onNewSessionCreate(options)
      } finally {
        setNewSessionLoading(false)
      }

      newSessionModalHandlers.close()
    },
    [newSessionModalHandlers, onNewSessionCreate]
  )

  return (
    <>
      <HeaderNewSessionModal
        opened={isNewSessionModalOpened}
        loading={isNewSessionLoading}
        onClose={newSessionModalHandlers.close}
        onCreate={handleNewSessionCreate}
        maxRequestBodySize={(!!appSettings && appSettings.maxRequestBodySize) || null}
      />
      <HeaderHelpModal
        opened={isHelpModalOpened}
        onClose={helpModalHandlers.close}
        webHookUrl={webHookUrl}
        sessionTTLSec={(!!appSettings && appSettings.sessionTTLSec) || null}
        maxBodySizeBytes={(!!appSettings && appSettings.maxRequestBodySize) || null}
        maxRequestsPerSession={(!!appSettings && appSettings.setMaxRequestsPerSession) || null}
      />

      <Group h="100%" px="md" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Burger opened={isBurgerOpened} onClick={onBurgerClick} hiddenFrom="sm" size="sm" />
        <Group>
          <Image
            src={LogoTextSvg}
            style={{ height: 20 }}
            title={currentVersion ? 'v' + currentVersion.toString() : undefined}
          />
          <Button.Group visibleFrom="md">
            <Button
              variant="default"
              size="xs"
              leftSection={<IconHelpHexagonFilled size="1.3em" />}
              onClick={helpModalHandlers.open}
            >
              Help
            </Button>

            {isUpdateAvailable && !!latestVersion ? (
              <Button
                variant="default"
                size="xs"
                leftSection={<IconRefreshAlert size="1.3em" />}
                component={Link}
                to={__LATEST_RELEASE_LINK__}
                rel="preload"
                target="_blank"
              >
                Update available {!!latestVersion && <>(v{latestVersion.toString()})</>}
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
                <Stack>
                  <Checkbox
                    checked={settings.autoNavigateToNewRequest}
                    onChange={(event) => updateSettings({ autoNavigateToNewRequest: event.target.checked })}
                    label="Automatically navigate to the new request"
                  />
                  <Checkbox
                    checked={settings.showRequestDetails}
                    onChange={(event) => updateSettings({ showRequestDetails: event.target.checked })}
                    label="Display request details"
                  />
                  <Checkbox
                    checked={settings.showNativeRequestNotifications}
                    onChange={(event) => {
                      if (browserNotificationsGranted) {
                        updateSettings({ showNativeRequestNotifications: event.target.checked })
                      } else {
                        askForBrowserNotifications().then((granted) => {
                          updateSettings({ showNativeRequestNotifications: granted && !event.target.checked })
                        })
                      }
                    }}
                    label={
                      <>
                        <Text size="sm">Use native notifications for new requests (instead of the in-app ones)</Text>
                        {!browserNotificationsGranted && (
                          <Text size="sm" c="dimmed" fw={700}>
                            Permission required
                          </Text>
                        )}
                      </>
                    }
                  />
                </Stack>
              </Popover.Dropdown>
            </Popover>

            {sessions.length > 1 && (
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
                    <Stack gap="xs" pb="0.25em">
                      {!!onSessionSwitch && (
                        <Select
                          label="Switch to a different webhook"
                          placeholder="Select a webhook ID to switch to"
                          comboboxProps={{ withinPortal: false }}
                          checkIconPosition="right"
                          data={sessions}
                          value={sID}
                          onChange={(switchTo) => {
                            if (switchTo) {
                              onSessionSwitch(switchTo)
                            }
                          }}
                        />
                      )}
                      {!!onSessionDestroy && !!sID && (
                        <Button
                          variant="light"
                          size="compact-sm"
                          leftSection={<IconGrave2 size="1.1em" />}
                          color="red"
                          disabled={!sID}
                          onClick={() => onSessionDestroy(sID)}
                        >
                          Destroy this webhook
                        </Button>
                      )}
                    </Stack>
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
              disabled={!webHookUrl}
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

import type { SemVer } from 'semver'
import { Burger, Button, Checkbox, Group, Image, Popover, Stack } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { useClipboard, useDisclosure } from '@mantine/hooks'
import {
  IconAdjustmentsAlt,
  IconBrandGithubFilled,
  IconCirclePlusFilled,
  IconCopy,
  IconHelpHexagonFilled,
  IconRefreshAlert,
} from '@tabler/icons-react'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useUISettings } from '~/shared'
import HeaderHelpModal from './header-help-modal'
import HeaderNewSessionModal, { type NewSessionOptions } from './header-new-session-modal'
import LogoTextSvg from '~/assets/togo-text.svg'

export default function Header({
  currentVersion,
  latestVersion,
  appSettings = null,
  webHookUrl = null,
  isBurgerOpened = false,
  onBurgerClick = () => {},
  onNewSessionCreate = () => Promise.reject(),
}: {
  currentVersion: SemVer | null
  latestVersion: SemVer | null
  appSettings: {
    setMaxRequestsPerSession: number
    maxRequestBodySize: number
    sessionTTLSec: number
  } | null
  webHookUrl: URL | null
  isBurgerOpened: boolean
  onBurgerClick: () => void
  onNewSessionCreate?: (_: NewSessionOptions) => Promise<void>
}): React.JSX.Element {
  const clipboard = useClipboard({ timeout: 500 })
  const { settings, updateSettings } = useUISettings()
  const [isUpdateAvailable, setIsUpdateAvailable] = useState<boolean>(false)
  const [isNewSessionModalOpened, newSessionModalHandlers] = useDisclosure(false)
  const [isHelpModalOpened, helpModalHandlers] = useDisclosure(false)
  const [isNewSessionLoading, setNewSessionLoading] = useState<boolean>(false)

  useEffect(() => {
    if (currentVersion && latestVersion) {
      setIsUpdateAvailable(currentVersion.compare(latestVersion) === -1)
    }
  }, [currentVersion, latestVersion])

  const handleCpyWebhookUrl = () => {
    if (webHookUrl) {
      clipboard.copy(webHookUrl.toString())

      notify.show({
        title: 'Webhook URL copied',
        message: 'The URL has been copied to your clipboard.',
        color: 'lime',
        autoClose: 3000,
      })
    }
  }

  const handleNewSessionCreate = async (options: NewSessionOptions) => {
    setNewSessionLoading(true)

    try {
      await onNewSessionCreate(options)
    } finally {
      setNewSessionLoading(false)
    }

    newSessionModalHandlers.close()
  }

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
            <Popover width={200} position="bottom" withArrow shadow="md">
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
                </Stack>
              </Popover.Dropdown>
            </Popover>

            <Button
              leftSection={<IconCopy size="1.2em" />}
              variant="filled"
              color="lime"
              onClick={handleCpyWebhookUrl}
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

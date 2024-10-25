import type { SemVer } from 'semver'
import { Burger, Button, Group, Image } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import { useClipboard, useDisclosure } from '@mantine/hooks'
import {
  IconBrandGithubFilled,
  IconCirclePlusFilled,
  IconCopy,
  IconHelpHexagonFilled,
  IconRefreshAlert,
} from '@tabler/icons-react'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import NewSessionModal, { type NewSessionOptions } from './new-session-modal'
import HelpModal from './help-modal'
import LogoTextSvg from '~/assets/togo-text.svg'

export default function Header({
  currentVersion,
  latestVersion,
  maxRequestBodySize = 0,
  sessionTTLSec = 0,
  maxBodySizeBytes = 0,
  webHookUrl = undefined,
  isBurgerOpened = false,
  onBurgerClick = () => {},
  onNewSessionCreate = () => Promise.reject(),
}: {
  currentVersion: SemVer | null
  latestVersion: SemVer | null
  maxRequestBodySize?: number
  sessionTTLSec?: number
  maxBodySizeBytes?: number
  webHookUrl?: URL
  isBurgerOpened: boolean
  onBurgerClick: () => void
  onNewSessionCreate?: (_: NewSessionOptions) => Promise<void>
}): React.JSX.Element {
  const clipboard = useClipboard({ timeout: 500 })
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
      <NewSessionModal
        opened={isNewSessionModalOpened}
        loading={isNewSessionLoading}
        onClose={newSessionModalHandlers.close}
        onCreate={handleNewSessionCreate}
        maxRequestBodySize={maxRequestBodySize}
      />
      <HelpModal
        opened={isHelpModalOpened}
        onClose={helpModalHandlers.close}
        webHookUrl={webHookUrl}
        sessionTTLSec={sessionTTLSec}
        maxBodySizeBytes={maxBodySizeBytes}
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
            {isUpdateAvailable && latestVersion ? (
              <Button
                variant="default"
                size="xs"
                leftSection={<IconRefreshAlert size="1.3em" />}
                component={Link}
                to={__LATEST_RELEASE_LINK__}
                rel="preload"
                target="_blank"
              >
                Update available {latestVersion && <>(v{latestVersion.toString()})</>}
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
            <Button
              leftSection={<IconCopy size="1.2em" />}
              variant="gradient"
              gradient={{ from: 'teal', to: 'lime', deg: -90 }}
              onClick={handleCpyWebhookUrl}
            >
              Copy Webhook URL
            </Button>
            <Button
              leftSection={<IconCirclePlusFilled size="1.3em" />}
              variant="gradient"
              gradient={{ from: 'blue', to: 'cyan', deg: 90 }}
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

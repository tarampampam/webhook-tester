import { Blockquote, Grid, List, Loader, Stack, Text } from '@mantine/core'
import { IconInfoHexagon } from '@tabler/icons-react'
import React from 'react'
import { useActiveSessionID } from '~/routing'
import { humanizeBytes, L10nKey as k, useAppConfig, useL10n, useSessions, useWebhookURL } from '~/shared'
import { CodeSnippets } from './components/code-snippets'
import { SessionProps } from './components/session-props'
import { ShellExamples } from './components/shell-examples'
import { WebhookURL } from './components/webhook-url'

export const SessionScreen = (): React.JSX.Element => {
  const { webhookURL } = useWebhookURL()
  const { sessions } = useSessions()
  const sID = useActiveSessionID()
  const { config } = useAppConfig()
  const { t, formatDuration } = useL10n()

  const session = sID && sessions.has(sID) ? sessions.get(sID) : null
  const sessionTTL = config?.limits.sessionTTL ? formatDuration(config.limits.sessionTTL) : null
  const maxBodySize = config?.limits.maxRequestBodySize ? humanizeBytes(config.limits.maxRequestBodySize) : null
  const maxRequests = config?.limits.maxRequests ?? null

  return (
    <Grid gap="md">
      <Grid.Col span={{ base: 12, lg: 9 }}>
        {(webhookURL && (
          <Stack gap="xs">
            <Text>{t(k.webhookTesterUsedFor)}.</Text>
            <Text fw="bolder" span>
              {t(k.yourUniqueWebhookUrl)}:
            </Text>
            <WebhookURL url={webhookURL} />

            <Text>{t(k.noNeedToRefresh)}</Text>

            {(sessionTTL || maxBodySize || maxRequests) && (
              <Blockquote color="yellow" p="md" m="md" icon={<IconInfoHexagon size="1em" stroke={1.5} />} iconSize={30}>
                <List fz="sm">
                  {sessionTTL && <List.Item>{t(k.sessionWillExpireAfter, { duration: sessionTTL })}</List.Item>}
                  {maxBodySize && <List.Item>{t(k.maxRequestBodySizeLimit, { size: maxBodySize })}</List.Item>}
                  {maxRequests && <List.Item>{t(k.maxRequestsLimit, { limit: maxRequests })}</List.Item>}
                </List>
              </Blockquote>
            )}

            <Text fz="0.85em">{t(k.sendSimpleRequestInShell)}:</Text>
            <ShellExamples url={webhookURL} />

            <Text fz="0.85em">{t(k.codeSnippets)}:</Text>
            <CodeSnippets url={webhookURL} />
          </Stack>
        )) || <PleaseWait />}
      </Grid.Col>
      <Grid.Col span="auto">
        {(!!session && (
          <Stack gap="xs">
            <Text ta="right">{t(k.webhookOptions)}:</Text>
            <SessionProps session={session} />
          </Stack>
        )) || <PleaseWait />}
      </Grid.Col>
    </Grid>
  )
}

const PleaseWait = (): React.JSX.Element => {
  const { t } = useL10n()

  return (
    <Stack h="100%" align="center" justify="center" gap={0}>
      <Loader color="cyan" mb="md" />
      <Text fz="0.7em" c="dimmed">
        {t(k.pleaseWait)}
      </Text>
    </Stack>
  )
}

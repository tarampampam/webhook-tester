import { Grid, Loader, Stack, Text } from '@mantine/core'
import React from 'react'
import { useActiveSessionID } from '~/routing'
import { L10nKey, useL10n, useSessions, useWebhookURL } from '~/shared'
import { CodeSnippets } from './components/code-snippets'
import { SessionProps } from './components/session-props'
import { ShellExamples } from './components/shell-examples'
import { WebhookURL } from './components/webhook-url'

export const SessionScreen = (): React.JSX.Element => {
  const { webhookURL } = useWebhookURL()
  const { sessions } = useSessions()
  const sID = useActiveSessionID()
  const { t } = useL10n()

  const session = sID && sessions.has(sID) ? sessions.get(sID) : null

  return (
    <Grid gap="md">
      <Grid.Col span={{ base: 12, lg: 9 }}>
        {(webhookURL && (
          <Stack gap="xs">
            <Text fw="bolder">{t(L10nKey.yourUniqueWebhookUrl)}:</Text>
            <WebhookURL url={webhookURL} />
            <Text fz="0.85em">{t(L10nKey.sendSimpleRequestInShell)}:</Text>
            <ShellExamples url={webhookURL} />
            <Text fz="0.85em">{t(L10nKey.codeSnippets)}:</Text>
            <CodeSnippets url={webhookURL} />
          </Stack>
        )) || <PleaseWait />}
      </Grid.Col>
      <Grid.Col span="auto">
        {(!!session && (
          <Stack gap="xs">
            <Text ta="right">{t(L10nKey.webhookOptions)}:</Text>
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
        {t(L10nKey.pleaseWait)}
      </Text>
    </Stack>
  )
}

import { Stack, Text } from '@mantine/core'
import React from 'react'
import { L10nKey, useL10n, useWebhookURL } from '~/shared'
import { CodeSnippets } from './components/code-snippets.tsx'
import { ShellExamples } from './components/shell-examples'
import { WebhookURL } from './components/webhook-url'

export const SessionScreen = (): React.JSX.Element => {
  const { webhookURL } = useWebhookURL()
  const { t } = useL10n()

  return (
    <>
      {webhookURL && (
        <Stack gap="xs">
          <Text>{t(L10nKey.yourUniqueWebhookUrl)}:</Text>
          <WebhookURL url={webhookURL} />
          <Text>{t(L10nKey.sendSimpleRequestInShell)}:</Text>
          <ShellExamples url={webhookURL} />
          <Text>{t(L10nKey.codeSnippets)}:</Text>
          <CodeSnippets url={webhookURL} />
        </Stack>
      )}
    </>
  )
}

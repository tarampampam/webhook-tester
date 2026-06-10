import type React from 'react'
import { CodeHighlight } from '@mantine/code-highlight'
import { Divider, Modal, Text } from '@mantine/core'
import { useAppConfig, useWebhookURL } from '~/shared'

export const HelpModal = ({ opened, onClose }: { opened: boolean; onClose: () => void }): React.JSX.Element => {
  const { webhookURL } = useWebhookURL()
  const { config } = useAppConfig()

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      size="xl"
      overlayProps={{
        backgroundOpacity: 0.55,
        blur: 3,
      }}
      title={
        <Text size="lg" fw={700}>
          What is Webhook Tester?
        </Text>
      }
      centered
    >
      <Text my="md">
        Webhook Tester lets you easily test webhooks and other HTTP requests. Here&apos;s your unique URL:
      </Text>

      <CodeHighlight code={webhookURL ? webhookURL.toString() : '...'} language="bash" w="100%" my="md" />

      <Text my="md">Any requests sent to this URL are instantly logged here &mdash; no need to refresh!</Text>

      <Divider my="md" />

      <Text my="md">To specify a status code in the response, append it to the URL, like so:</Text>

      <CodeHighlight code={webhookURL ? webhookURL.toString() + '/404' : '.../404'} language="bash" w="100%" my="md" />

      <Text my="md">This way, the URL will respond with a 404 status.</Text>

      <Text my="md">
        Feel free to bookmark this page to revisit the request details at any time.
        {!!config?.limits.sessionTTL &&
          config.limits.sessionTTL > 0 &&
          ` Requests and tokens for this URL expire after ${config.limits.sessionTTL / 60 / 60 / 24} days of inactivity.`}
        {!!config?.limits.maxRequestBodySize &&
          config.limits.maxRequestBodySize > 0 &&
          ` The maximum size for incoming requests is ${bytesToKilobytes(config.limits.maxRequestBodySize)} KiB.`}
        {!!config?.limits.maxRequests &&
          config.limits.maxRequests > 0 &&
          ` The maximum number of requests per session is ${config.limits.maxRequests}.`}
      </Text>
    </Modal>
  )
}

const bytesToKilobytes = (bytes: number): number => {
  if (isFinite(bytes)) {
    return Number((bytes / 1024).toFixed(1))
  }

  return 0
}

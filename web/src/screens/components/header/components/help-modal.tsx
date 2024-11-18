import type React from 'react'
import { CodeHighlight } from '@mantine/code-highlight'
import { Divider, Modal, Text } from '@mantine/core'
import { useData, useSettings } from '~/shared'

export const HelpModal: React.FC<{
  opened: boolean
  onClose: () => void
}> = ({ opened, onClose }) => {
  const { webHookUrl } = useData()
  const { sessionTTLSec, maxRequestBodySize, maxRequestsPerSession } = useSettings()

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

      <CodeHighlight code={webHookUrl ? webHookUrl.toString() : '...'} language="bash" w="100%" my="md" />

      <Text my="md">Any requests sent to this URL are instantly logged here &mdash; no need to refresh!</Text>

      <Divider my="md" />

      <Text my="md">To specify a status code in the response, append it to the URL, like so:</Text>

      <CodeHighlight code={webHookUrl ? webHookUrl.toString() + '/404' : '.../404'} language="bash" w="100%" my="md" />

      <Text my="md">This way, the URL will respond with a 404 status.</Text>

      <Text my="md">
        Feel free to bookmark this page to revisit the request details at any time.
        {!!sessionTTLSec &&
          sessionTTLSec > 0 &&
          ` Requests and tokens for this URL expire after ${sessionTTLSec / 60 / 60 / 24} days of inactivity.`}
        {!!maxRequestBodySize &&
          maxRequestBodySize > 0 &&
          ` The maximum size for incoming requests is ${bytesToKilobytes(maxRequestBodySize)} KiB.`}
        {!!maxRequestsPerSession &&
          maxRequestsPerSession > 0 &&
          ` The maximum number of requests per session is ${maxRequestsPerSession}.`}
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

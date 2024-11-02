import { CodeHighlight } from '@mantine/code-highlight'
import { Divider, Modal, Text, Title } from '@mantine/core'
import React from 'react'

export default function HeaderHelpModal({
  opened,
  onClose,
  webHookUrl = null,
  sessionTTLSec = 0,
  maxBodySizeBytes = 0,
  maxRequestsPerSession = 0,
}: {
  opened: boolean
  onClose: () => void
  webHookUrl: URL | null
  sessionTTLSec: number | null
  maxBodySizeBytes: number | null
  maxRequestsPerSession: number | null
}): React.JSX.Element {
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      size="lg"
      overlayProps={{
        backgroundOpacity: 0.55,
        blur: 3,
      }}
      title={<Title size="h3">What is Webhook Tester?</Title>}
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
        {!!maxBodySizeBytes &&
          maxBodySizeBytes > 0 &&
          ` The maximum size for incoming requests is ${bytesToKilobytes(maxBodySizeBytes)} KiB.`}
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

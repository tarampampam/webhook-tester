import { Button, Checkbox, Group, Modal, NumberInput, Space, Text, Textarea, Title } from '@mantine/core'
import { useSessionStorage } from '@mantine/hooks'
import { IconCodeAsterisk, IconHeading, IconHourglassHigh, IconVersions } from '@tabler/icons-react'
import React, { useState } from 'react'

const limits = {
  statusCode: { min: 100, max: 530 },
  responseHeaders: { maxCount: 10, minNameLen: 1, maxNameLen: 40, maxValueLen: 2048 },
  delay: { min: 0, max: 30 },
}

export type NewSessionOptions = {
  statusCode: number
  headers: Array<{ name: string; value: string }>
  delay: number
  responseBody: string
  destroyCurrentSession: boolean
}

const storageKeyPrefix = 'webhook-tester-v2-new-session'

export default function NewSessionModal({
  opened,
  loading = false,
  onClose,
  onCreate,
  maxRequestBodySize = 10240,
}: {
  opened: boolean
  loading?: boolean
  onClose: () => void
  onCreate: (_: NewSessionOptions) => void
  maxRequestBodySize?: number
}): React.JSX.Element {
  const [statusCode, setStatusCode] = useSessionStorage<number>({
    key: `${storageKeyPrefix}-status-code`,
    defaultValue: 200,
  })
  const [headersList, setHeadersList] = useSessionStorage<string>({
    key: `${storageKeyPrefix}-headers-list`,
    defaultValue: 'Content-Type: application/json\nServer: WebhookTester',
  })
  const [delay, setDelay] = useSessionStorage<number>({
    key: `${storageKeyPrefix}-delay`,
    defaultValue: 0,
  })
  const [responseBody, setResponseBody] = useSessionStorage<string>({
    key: `${storageKeyPrefix}-response-body`,
    defaultValue: '',
  })
  const [destroyCurrentSession, setDestroyCurrentSession] = useSessionStorage<boolean>({
    key: `${storageKeyPrefix}-destroy-current-session`,
    defaultValue: true,
  })

  const [wrongStatusCode, setWrongStatusCode] = useState<boolean>(false)
  const [wrongDelay, setWrongDelay] = useState<boolean>(false)
  const [wrongResponseBody, setWrongResponseBody] = useState<boolean>(false)

  const handleCreate = () => {
    // validate all the fields
    if (statusCode < limits.statusCode.min || statusCode > limits.statusCode.max) {
      setWrongStatusCode(true)

      return
    } else {
      setWrongStatusCode(false)
    }

    const headers = headersList
      .split('\n') // split by each line
      .map((line) => {
        const [name, ...valueParts] = line.split(': ')
        const value = valueParts.join(': ') // join in case of additional colons in value

        return { name: name.trim(), value: value.trim() }
      })
      .filter((header) => header.name && header.value) // remove empty headers
      .filter((header) => header.name.length >= limits.responseHeaders.minNameLen) // filter by min name length
      .filter((header) => header.name.length <= limits.responseHeaders.maxNameLen) // filter by max name length
      .filter((header) => header.value.length <= limits.responseHeaders.maxValueLen) // filter by max value length
      .slice(0, limits.responseHeaders.maxCount)

    if (delay < limits.delay.min || delay > limits.delay.max) {
      setWrongDelay(true)

      return
    } else {
      setWrongDelay(false)
    }

    if (maxRequestBodySize > 0 && responseBody.length > maxRequestBodySize) {
      setWrongResponseBody(true)

      return
    } else {
      setWrongResponseBody(false)
    }

    onCreate({ statusCode, headers, delay, responseBody, destroyCurrentSession })
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      size="md"
      overlayProps={{
        backgroundOpacity: 0.55,
        blur: 3,
      }}
      title={<Title size="h3">Configure URL</Title>}
      centered
    >
      <Text size="xs">
        You have the ability to customize how your URL will respond by changing the status code, headers, response delay
        and the content.
      </Text>
      <Space h="sm" />
      <NumberInput
        my="sm"
        label="Default status code"
        description="The default status code for the URL"
        placeholder="200"
        allowDecimal={false}
        leftSection={<IconCodeAsterisk />}
        min={limits.statusCode.min}
        max={limits.statusCode.max}
        error={wrongStatusCode}
        disabled={loading}
        value={statusCode}
        onChange={(v: string | number): void => setStatusCode(typeof v === 'string' ? parseInt(v, 10) : v)}
      />
      <Textarea
        my="sm"
        label="Response headers"
        description={`The list of headers to include in the response (one per line, max ${limits.responseHeaders.maxCount})`}
        placeholder={'Content-Type: application/json\nServer: WebhookTester\nX-Request-Id: 3C27:3A7ABF:250756C'}
        leftSection={<IconHeading />}
        styles={{ input: { fontFamily: 'monospace', fontSize: '0.9em' } }}
        minRows={2}
        maxRows={10}
        disabled={loading}
        value={headersList}
        onChange={(e) => setHeadersList(e.currentTarget.value)}
        autosize
      />
      <NumberInput
        my="sm"
        label="Response delay"
        description="The delay in seconds before the response is sent"
        placeholder="0"
        allowDecimal={false}
        leftSection={<IconHourglassHigh />}
        min={limits.delay.min}
        max={limits.delay.max}
        error={wrongDelay}
        disabled={loading}
        value={delay}
        onChange={(v: string | number): void => setDelay(typeof v === 'string' ? parseInt(v, 10) : v)}
      />
      <Textarea
        my="sm"
        label="Response body"
        description={`The content of the response${maxRequestBodySize > 0 && ` (max ${new Intl.NumberFormat().format(maxRequestBodySize)} characters)`}`}
        placeholder={'{"message": "Hello, World!"}'}
        leftSection={<IconVersions />}
        styles={{ input: { fontFamily: 'monospace', fontSize: '0.9em' } }}
        minRows={1}
        maxRows={15}
        error={wrongResponseBody}
        disabled={loading}
        value={responseBody}
        onChange={(e) => setResponseBody(e.currentTarget.value)}
        autosize
      />
      <Group mt="xl" justify="space-between">
        <Checkbox
          my="sm"
          label="Destroy current session"
          disabled={loading}
          checked={destroyCurrentSession}
          onChange={(e) => setDestroyCurrentSession(e.currentTarget.checked)}
        />
        <Button
          variant="filled"
          color="green"
          size="md"
          radius="xl"
          onClick={handleCreate}
          loading={loading}
          data-autofocus
        >
          Create
        </Button>
      </Group>
    </Modal>
  )
}

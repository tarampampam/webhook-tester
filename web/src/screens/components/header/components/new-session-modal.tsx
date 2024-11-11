import React, { useState } from 'react'
import { Button, Checkbox, Group, Modal, NumberInput, Space, Text, Textarea, Title } from '@mantine/core'
import { IconCodeAsterisk, IconHeading, IconHourglassHigh, IconVersions } from '@tabler/icons-react'
import { notifications as notify } from '@mantine/notifications'
import { useNavigate } from 'react-router-dom'
import { useStorage, UsedStorageKeys, type StorageArea, useSettings, useData } from '~/shared'
import { pathTo, RouteIDs } from '~/routing'

/** Controls for the new session modal */
const controls = {
  // status code
  code: {
    def: 200,
    limits: { min: 200, max: 530 },
    key: UsedStorageKeys.NewSessionStatusCode,
    area: 'session' satisfies StorageArea as StorageArea,
  },
  // response headers
  head: {
    def: 'Content-Type: application/json\nServer: WebhookTester',
    limits: { maxCount: 10, minNameLen: 1, maxNameLen: 40, maxValueLen: 2048 },
    key: UsedStorageKeys.NewSessionHeadersList,
    area: 'session' satisfies StorageArea as StorageArea,
  },
  // response delay
  delay: {
    def: 0,
    limits: { min: 0, max: 30 },
    key: UsedStorageKeys.NewSessionSessionDelay,
    area: 'session' satisfies StorageArea as StorageArea,
  },
  // response body
  body: {
    def: '{"captured": true}',
    key: UsedStorageKeys.NewSessionResponseBody,
    area: 'session' satisfies StorageArea as StorageArea,
  },
  // destroy current session
  destroy: {
    def: false,
    key: UsedStorageKeys.NewSessionDestroyCurrentSession,
    area: 'session' satisfies StorageArea as StorageArea,
  },
}

/** Validation functions for the controls */
const validate: { [K in keyof typeof controls]: (v: unknown) => boolean } = {
  code: (v) => typeof v === 'number' && v >= controls.code.limits.min && v <= controls.code.limits.max,
  head: (v) => {
    if (typeof v !== 'string') {
      return false
    }

    const converted = headersTextToHeaders(v) // convert the text to headers

    return (
      converted.length <= controls.head.limits.maxCount && // check the count of headers
      converted.every(
        (header) =>
          header.name.length >= controls.head.limits.minNameLen && // check the name length (min)
          header.name.length <= controls.head.limits.maxNameLen && // check the name length (max)
          header.value.length <= controls.head.limits.maxValueLen // check the value length (max)
      )
    )
  },
  delay: (v) => typeof v === 'number' && v >= controls.delay.limits.min && v <= controls.delay.limits.max,
  body: (v) => typeof v === 'string',
  destroy: (v) => typeof v === 'boolean',
}

let count: number = 0

export const NewSessionModal: React.FC<{
  opened: boolean
  onClose: () => void
}> = ({ opened, onClose }) => {
  console.debug(`🖌 NewSessionModal rendering (${++count})`)

  const navigate = useNavigate()
  const { maxRequestBodySize: maxBodySize } = useSettings()
  const { session, newSession, destroySession, switchToSession } = useData()
  const [loading, setLoading] = useState<boolean>(false)

  const [status, setStatus] = useStorage<number>(controls.code.def, controls.code.key, controls.code.area)
  const [headers, setHeaders] = useStorage<string>(controls.head.def, controls.head.key, controls.head.area)
  const [delay, setDelay] = useStorage<number>(controls.delay.def, controls.delay.key, controls.delay.area)
  const [body, setBody] = useStorage<string>(controls.body.def, controls.body.key, controls.body.area)
  const [destroy, setDestroy] = useStorage<boolean>(controls.destroy.def, controls.destroy.key, controls.destroy.area)

  const [wrongStatusCode, setWrongStatusCode] = useState<boolean>(false)
  const [wrongHeaders, setWrongHeaders] = useState<boolean>(false)
  const [wrongDelay, setWrongDelay] = useState<boolean>(false)
  const [wrongResponseBody, setWrongResponseBody] = useState<boolean>(false)

  /** Handle the creation of a new session */
  const handleCreate = () => {
    // validate all the fields
    const validated: { [K in keyof typeof controls]: boolean } = {
      code: validate.code(status),
      head: validate.head(headers),
      delay: validate.delay(delay),
      body: validate.body(body) && (!maxBodySize || maxBodySize <= 0 || body.length <= maxBodySize),
      destroy: validate.destroy(destroy),
    }

    // set the error states
    setWrongStatusCode(!validated.code)
    setWrongHeaders(!validated.head)
    setWrongDelay(!validated.delay)
    setWrongResponseBody(!validated.body)

    // if any of the fields are invalid, return
    if (!Object.values(validated).every((v) => v)) {
      return
    }

    // cook the headers (convert text to an array and then to object)
    const respHeaders: { [k: string]: string } = Object.fromEntries(
      headersTextToHeaders(headers).map((h) => [h.name, h.value])
    )

    if (!session) {
      throw new Error('No active session')
    }

    // set the loading state
    setLoading(true)

    const id = notify.show({ title: 'Creating new WebHook', message: null, autoClose: false, loading: true })

    // create the new session
    newSession({
      statusCode: status,
      headers: Object.keys(respHeaders).length > 0 ? respHeaders : undefined,
      delay: delay > 0 ? delay : undefined,
      responseBody: body.trim().length > 0 ? new TextEncoder().encode(body) : undefined,
    })
      .then((opts) => {
        const [currentSID, newSID] = [session.sID, opts.sID]

        notify.update({
          id,
          title: 'A new WebHook has been created!',
          message: null,
          color: 'green',
          autoClose: 7000,
          loading: false,
        })

        if (destroy) {
          destroySession(currentSID).catch((err) => {
            notify.show({
              title: 'Failed to destroy current WebHook',
              message: String(err),
              color: 'red',
              autoClose: 5000,
            })
          })
        }

        switchToSession(newSID).then(() => navigate(pathTo(RouteIDs.SessionAndRequest, newSID)))
      })
      .catch((err) => {
        notify.update({
          id,
          title: 'Failed to create new WebHook',
          message: String(err),
          color: 'red',
          loading: false,
        })
      })
      .finally(() => setLoading(false))
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      size="md"
      overlayProps={{ backgroundOpacity: 0.55, blur: 3 }}
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
        min={controls.code.limits.min}
        max={controls.code.limits.max}
        error={wrongStatusCode}
        disabled={loading}
        value={status}
        onChange={(v: string | number): void => setStatus(typeof v === 'string' ? parseInt(v, 10) : v)}
      />
      <Textarea
        my="sm"
        label="Response headers"
        description={`The list of headers to include in the response (one per line, max ${controls.head.limits.maxCount})`}
        placeholder={'Content-Type: application/json\nServer: WebhookTester\nX-Request-Id: 3C27:3A7ABF:250756C'}
        leftSection={<IconHeading />}
        styles={{ input: { fontFamily: 'monospace', fontSize: '0.9em' } }}
        minRows={2}
        maxRows={10}
        error={wrongHeaders}
        disabled={loading}
        value={headers}
        onChange={(e) => setHeaders(e.currentTarget.value)}
        autosize
      />
      <NumberInput
        my="sm"
        label="Response delay"
        description="The delay in seconds before the response is sent"
        placeholder="0"
        allowDecimal={false}
        leftSection={<IconHourglassHigh />}
        min={controls.delay.limits.min}
        max={controls.delay.limits.max}
        error={wrongDelay}
        disabled={loading}
        value={delay}
        onChange={(v: string | number): void => setDelay(typeof v === 'string' ? parseInt(v, 10) : v)}
      />
      <Textarea
        my="sm"
        label="Response body"
        description={`The content of the response${!!maxBodySize && maxBodySize > 0 && ` (max ${new Intl.NumberFormat().format(maxBodySize)} characters)`}`}
        placeholder={'{"message": "Hello, World!"}'}
        leftSection={<IconVersions />}
        styles={{ input: { fontFamily: 'monospace', fontSize: '0.9em' } }}
        minRows={1}
        maxRows={15}
        error={wrongResponseBody}
        disabled={loading}
        value={body}
        onChange={(e) => setBody(e.currentTarget.value)}
        autosize
      />
      <Group mt="xl" justify="space-between">
        <Checkbox
          my="sm"
          label="Destroy current session"
          disabled={loading}
          checked={destroy}
          onChange={(e) => setDestroy(e.currentTarget.checked)}
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

/** Convert text to headers */
const headersTextToHeaders = (text: string): Array<{ name: string; value: string }> =>
  text
    .split('\n') // split by each line
    .map((line) => {
      const [name, ...valueParts] = line.split(': ')
      const value = valueParts.join(': ') // join in case of additional colons in value

      return { name: name.trim(), value: value.trim() }
    })
    .filter((header) => header.name && header.value) // remove empty headers

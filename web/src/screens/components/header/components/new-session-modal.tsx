import React, { useState, useEffect } from 'react'
import { Button, Checkbox, Group, Modal, NumberInput, Space, Text, Textarea, Select, MultiSelect, TagsInput } from '@mantine/core'
import { IconCodeAsterisk, IconHeading, IconHourglassHigh, IconVersions, IconLink, IconPlayerPlay } from '@tabler/icons-react'
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
  // proxy urls
  proxyUrls: {
    def: [] as string[], // Default to empty array
    key: UsedStorageKeys.NewSessionProxyUrls, // Define this key in UsedStorageKeys
    area: 'session' satisfies StorageArea as StorageArea,
    limits: { maxCount: 5, maxUrlLength: 2048 }, // Example limits
  },
  // proxy response mode
  proxyResponseMode: {
    def: 'app_response',
    key: UsedStorageKeys.NewSessionProxyResponseMode, // Define this key in UsedStorageKeys
    area: 'session' satisfies StorageArea as StorageArea,
  },
  // destroy current session
  destroy: {
    def: true,
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

    const raw = headersTextToHeaders(v) // convert the text to headers

    return (
      raw.length <= controls.head.limits.maxCount && // check the count of headers
      raw.every(
        (h) =>
          h.name.length >= controls.head.limits.minNameLen && // check the name length (min)
          h.name.length <= controls.head.limits.maxNameLen && // check the name length (max)
          h.value.length <= controls.head.limits.maxValueLen && // check the value length (max)
          /^[a-zA-Z0-9-]+$/i.test(h.name) &&
          /^[^\r\n]*$/i.test(h.value) && // check the header name and value format
          h.name.trim().length > 0 &&
          h.value.trim().length > 0 // check the header name and value are not empty
      )
    )
  },
  delay: (v) => typeof v === 'number' && v >= controls.delay.limits.min && v <= controls.delay.limits.max,
  body: (v) => typeof v === 'string',
  proxyUrls: (v) => {
    if (!Array.isArray(v)) return false
    if (v.length > controls.proxyUrls.limits.maxCount) return false
    return v.every(
      (url) => {
        if (typeof url !== 'string' || url.length > controls.proxyUrls.limits.maxUrlLength) return false
        try {
          new URL(url) // Check if it's a valid URL
          return true
        } catch {
          return false
        }
      }
    )
  },
  proxyResponseMode: (v) => typeof v === 'string' && ['app_response', 'proxy_first_success'].includes(v),
  destroy: (v) => typeof v === 'boolean',
}

export const NewSessionModal: React.FC<{
  opened: boolean
  onClose: () => void
}> = ({ opened, onClose }) => {
  const navigate = useNavigate()
  const { maxRequestBodySize: maxBodySize } = useSettings()
  const { session, newSession, destroySession } = useData()
  const [loading, setLoading] = useState<boolean>(false)

  const [status, setStatus] = useStorage<number>(controls.code.def, controls.code.key, controls.code.area)
  const [headers, setHeaders] = useStorage<string>(controls.head.def, controls.head.key, controls.head.area)
  const [delay, setDelay] = useStorage<number>(controls.delay.def, controls.delay.key, controls.delay.area)
  const [body, setBody] = useStorage<string>(controls.body.def, controls.body.key, controls.body.area)
  const [proxyUrls, setProxyUrls] = useStorage<string[]>(controls.proxyUrls.def, controls.proxyUrls.key, controls.proxyUrls.area)
  const [proxyResponseMode, setProxyResponseMode] = useStorage<string>(controls.proxyResponseMode.def, controls.proxyResponseMode.key, controls.proxyResponseMode.area)
  const [destroy, setDestroy] = useStorage<boolean>(controls.destroy.def, controls.destroy.key, controls.destroy.area)

  const [wrongStatusCode, setWrongStatusCode] = useState<boolean>(false)
  const [wrongHeaders, setWrongHeaders] = useState<boolean>(false)
  const [wrongDelay, setWrongDelay] = useState<boolean>(false)
  const [wrongResponseBody, setWrongResponseBody] = useState<boolean>(false)
  const [wrongProxyUrls, setWrongProxyUrls] = useState<boolean>(false)
  const [wrongProxyResponseMode, setWrongProxyResponseMode] = useState<boolean>(false)
  const [createDisabled, setCreateDisabled] = useState<boolean>(
    wrongStatusCode || wrongHeaders || wrongDelay || wrongResponseBody || wrongProxyUrls || wrongProxyResponseMode
  )

  // watch the values and set the "wrong" state when the value changes
  useEffect(() => setWrongStatusCode(!validate.code(status)), [status])
  useEffect(() => setWrongHeaders(!validate.head(headers)), [headers])
  useEffect(() => setWrongDelay(!validate.delay(delay)), [delay])
  useEffect(() => {
    let bodyIsValid = validate.body(body) // validate the body

    // if max body size is set and the body is valid
    if (!!maxBodySize && bodyIsValid) {
      bodyIsValid = body.length <= maxBodySize // check the body length
    }

    setWrongResponseBody(!bodyIsValid)
  }, [body, maxBodySize])
  useEffect(() => setWrongProxyUrls(!validate.proxyUrls(proxyUrls)), [proxyUrls])
  useEffect(() => setWrongProxyResponseMode(!validate.proxyResponseMode(proxyResponseMode)), [proxyResponseMode])

  // disable the create button if any of the fields are invalid
  useEffect(
    () => setCreateDisabled(wrongStatusCode || wrongHeaders || wrongDelay || wrongResponseBody || wrongProxyUrls || wrongProxyResponseMode),
    [wrongStatusCode, wrongHeaders, wrongDelay, wrongResponseBody, wrongProxyUrls, wrongProxyResponseMode]
  )

  /** Handle the creation of a new session */
  const handleCreate = () => {
    // if any of the fields are invalid, return (kinda fuse)
    if (wrongStatusCode || wrongHeaders || wrongDelay || wrongResponseBody || wrongProxyUrls || wrongProxyResponseMode) {
      return
    }

    // cook the headers (convert text to an array and then to object)
    const respHeaders: { [k: string]: string } = Object.fromEntries(
      headersTextToHeaders(headers).map((h) => [h.name, h.value])
    )

    // set the loading state
    setLoading(true)

    const id = notify.show({ title: 'Creating new WebHook', message: null, autoClose: false, loading: true })

    // create the new session
    newSession({
      statusCode: status,
      headers: Object.keys(respHeaders).length > 0 ? respHeaders : undefined,
      delay: delay > 0 ? delay : undefined,
      responseBody: body.trim().length > 0 ? new TextEncoder().encode(body) : undefined,
      proxyUrls: proxyUrls.length > 0 ? proxyUrls.filter(url => url.trim() !== '') : undefined,
      proxyResponseMode: proxyResponseMode,
    })
      .then((opts) => {
        notify.update({
          id,
          title: 'A new WebHook has been created!',
          message: null,
          color: 'green',
          autoClose: 7000,
          loading: false,
        })

        if (destroy && !!session) {
          destroySession(session.sID)
            .then((slow) => slow())
            .catch((err) => {
              notify.show({
                title: 'Failed to destroy current WebHook',
                message: String(err),
                color: 'red',
                autoClose: 5000,
              })
            })
        }

        onClose()

        navigate(pathTo(RouteIDs.SessionAndRequest, opts.sID))
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
      title={
        <Text size="lg" fw={700}>
          Configure URL
        </Text>
      }
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
      <TagsInput
        my="sm"
        label="Proxy URLs"
        description={`Enter URLs to forward requests to (one per entry, max ${controls.proxyUrls.limits.maxCount})`}
        placeholder="https://example.com/webhook"
        leftSection={<IconLink />}
        data={proxyUrls}
        onChange={setProxyUrls}
        error={wrongProxyUrls}
        disabled={loading}
        maxTags={controls.proxyUrls.limits.maxCount}
        clearable
      />
      <Select
        my="sm"
        label="Proxy Response Mode"
        description="Determines how the webhook tester responds when proxying"
        leftSection={<IconPlayerPlay />}
        data={[
          { value: 'app_response', label: 'Application Response' },
          { value: 'proxy_first_success', label: 'Proxy First Success' },
        ]}
        value={proxyResponseMode}
        onChange={(value) => setProxyResponseMode(value || controls.proxyResponseMode.def)}
        error={wrongProxyResponseMode}
        disabled={loading}
        allowDeselect={false}
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
          disabled={createDisabled}
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
    .map((line) => line.trim()) // trim each line
    .filter((line) => line.length > 0) // filter out empty lines
    .map((line) => {
      const [name, ...valueParts] = line.split(': ')
      const value = valueParts.join(': ') // join in case of additional colons in value

      return { name: name.trim(), value: value.trim() }
    })

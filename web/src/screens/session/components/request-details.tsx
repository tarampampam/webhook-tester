import { CodeHighlight, CodeHighlightTabs } from '@mantine/code-highlight'
import { Badge, Code, Grid, Skeleton, Table, Text, Title, Tabs, Button } from '@mantine/core'
import { useInterval, useSessionStorage } from '@mantine/hooks'
import { IconBinary, IconDownload, IconLetterCase } from '@tabler/icons-react'
import dayjs from 'dayjs'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import type { Client } from '~/api'
import { storageKey, useUISettings } from '../../../shared'
import { methodToColor } from '../../../theme'

export default function RequestDetails({
  apiClient,
  sID,
  rID,
}: {
  apiClient: Client
  sID: string
  rID: string
}): React.JSX.Element {
  const [headersExpanded, setHeadersExpanded] = useSessionStorage<boolean>({
    key: storageKey('request-headers-expanded'),
    defaultValue: false,
  })
  const { settings: uiSettings } = useUISettings()
  const [loading, setLoading] = React.useState<boolean>(true)
  const [url, setUrl] = React.useState<URL | null>(null)
  const [method, setMethod] = React.useState<string | null>(null)
  const [clientAddress, setClientAddress] = React.useState<string | null>(null)
  const [capturedAt, setCapturedAt] = React.useState<Date | null>(null)
  const [elapsedTime, setElapsedTime] = useState<string | null>(null)
  const [payload, setPayload] = React.useState<Uint8Array | null>(null)
  const [headers, setHeaders] = React.useState<ReadonlyArray<{ name: string; value: string }> | null>(null)

  // automatically update the elapsed time
  {
    useEffect(() => setElapsedTime(capturedAt ? dayjs(capturedAt).fromNow() : null), [capturedAt, setElapsedTime])
    const interval = useInterval(() => setElapsedTime(capturedAt ? dayjs(capturedAt).fromNow() : null), 1000)

    useEffect((): (() => void) => {
      interval.start()

      return interval.stop
    }, []) // eslint-disable-line react-hooks/exhaustive-deps
  }

  useEffect(() => {
    setLoading(true)

    apiClient
      .getSessionRequest(sID, rID)
      .then((request) => {
        setUrl(request.url)
        setMethod(request.method)
        setClientAddress(request.clientAddress)
        setCapturedAt(request.capturedAt)
        setPayload(request.requestPayload)
        setHeaders(request.headers)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [apiClient, sID, rID])

  const formatUrlToDisplay = (url: URL): string => {
    const { pathname, search, hash } = url

    return `${pathname}${search}${hash}`
  }

  const payloadText = (data: Uint8Array | null): string => {
    if (!data) {
      return ''
    }

    const asString = new TextDecoder('utf-8').decode(data)

    // try to format as json
    try {
      return JSON.stringify(JSON.parse(asString), undefined, 2)

      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (_) {
      // wrong json
    }

    return asString
  }

  const handleDownload = (data: Uint8Array, fileName: string): void => {
    const blob = new Blob([data], { type: 'application/octet-stream' })
    const url = URL.createObjectURL(blob)
    const downloadLink = document.createElement('a')

    downloadLink.href = url
    downloadLink.download = fileName
    downloadLink.click()

    URL.revokeObjectURL(url)
  }

  return (
    <Grid>
      {uiSettings.showRequestDetails && (
        <>
          <Grid.Col span={{ base: 12, md: 6 }}>
            <Title order={4} mb="md">
              Request details
            </Title>
            <Table my="md" withRowBorders={false} verticalSpacing="0.2em" highlightOnHover>
              <Table.Tbody>
                <Table.Tr>
                  <Table.Td ta="right" w="15%">
                    URL
                  </Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="80%" />) ||
                      (url && <Code>{formatUrlToDisplay(url)}</Code>) || <>...</>}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">Method</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="15%" />) || (
                      <Badge color={methodToColor(method ?? '')}>{method}</Badge>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">From</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="20%" />) || (
                      <Link to={'https://who.is/whois-ip/ip-address/' + clientAddress} target="_blank" rel="noreferrer">
                        {clientAddress}
                      </Link>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">When</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="45%" />) || (
                      <>
                        {capturedAt && <>{dayjs(capturedAt).format('YYYY-MM-DD HH:mm:ss')}</>}
                        {elapsedTime && (
                          <Text pl="0.3em" span>
                            ({elapsedTime})
                          </Text>
                        )}
                      </>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">Size</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="15%" />) || <>{payload?.length} bytes</>}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">ID</Table.Td>
                  <Table.Td>{(loading && <Skeleton radius="xl" h="sm" w="50%" />) || <>{rID}</>}</Table.Td>
                </Table.Tr>
              </Table.Tbody>
            </Table>
          </Grid.Col>

          <Grid.Col span={{ base: 12, md: 6 }}>
            <Title order={4} mb="md">
              HTTP headers
            </Title>
            {(loading && <Skeleton radius="md" h="10em" w="100%" />) ||
              (!!headers && (
                <CodeHighlightTabs
                  code={{
                    fileName: 'headers.txt',
                    code: headers.map(({ name, value }) => `${name}: ${value}`).join('\n'),
                    language: 'bash',
                  }}
                  expandCodeLabel="Show all headers"
                  maxCollapsedHeight="10em"
                  expanded={headersExpanded}
                  onExpandedChange={setHeadersExpanded}
                  withExpandButton
                  withCopyButton
                />
              ))}
          </Grid.Col>
        </>
      )}

      <Grid.Col span={12}>
        <Title order={4} mb="md">
          Request body
          {!!payload && payload.length > 0 && (
            <Button
              variant="light"
              color="indigo"
              size="compact-sm"
              ml="sm"
              leftSection={<IconDownload size="1.2em" />}
              onClick={() => handleDownload(payload, `${rID}.bin`)}
            >
              Download
            </Button>
          )}
        </Title>
        <Tabs variant="default" defaultValue={TabsList.Text} keepMounted={false}>
          <Tabs.List>
            <Tabs.Tab value={TabsList.Text} leftSection={<IconLetterCase />} color="blue">
              Text
            </Tabs.Tab>
            <Tabs.Tab value={TabsList.Binary} leftSection={<IconBinary />} color="teal">
              Binary (TODO)
            </Tabs.Tab>
          </Tabs.List>
          <Tabs.Panel value={TabsList.Text}>
            <CodeHighlight code={payloadText(payload)} language="json" />
          </Tabs.Panel>
          <Tabs.Panel value={TabsList.Binary}>Binary</Tabs.Panel>
        </Tabs>
      </Grid.Col>
    </Grid>
  )
}

enum TabsList {
  Text = 'Text',
  Binary = 'Binary',
}

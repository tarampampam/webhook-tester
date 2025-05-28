import React, { useEffect, useState } from 'react'
import { CodeHighlightTabs } from '@mantine/code-highlight'
import { Accordion, Badge, Button, Flex, Grid, Skeleton, Table, Tabs, Text, Title, Paper, Group } from '@mantine/core'
import { useInterval } from '@mantine/hooks'
import { Link } from 'react-router-dom'
import { IconBinary, IconDownload, IconLetterCase } from '@tabler/icons-react'
import dayjs from 'dayjs'
import { useData, UsedStorageKeys, useSettings, useStorage } from '~/shared'
import { methodToColor } from '~/theme'
import { ViewHex, ViewText } from './components'

export const RequestDetails: React.FC<{ loading?: boolean }> = ({ loading = false }) => {
  const { session, request } = useData()
  const { showRequestDetails } = useSettings()

  const [headersExpanded, setHeadersExpanded] = useStorage<boolean>(false, UsedStorageKeys.RequestDetailsHeadersExpand)
  const [elapsedTime, setElapsedTime] = useState<string | null>(null)
  const [contentType, setContentType] = useState<string | null>(null)
  const [payload, setPayload] = useState<Uint8Array | null>(null)

  useEffect(
    () => setContentType(request?.headers.find(({ name }) => name.toLowerCase() === 'content-type')?.value ?? null),
    [request]
  )

  // automatically update the payload
  useEffect(() => {
    request?.payload?.then((data) => setPayload(data))
  }, [request, request?.payload])

  // automatically update the elapsed time
  useEffect(
    () => setElapsedTime(request?.capturedAt ? dayjs(request?.capturedAt).fromNow() : null),
    [request?.capturedAt, setElapsedTime]
  )
  const interval = useInterval(
    () => setElapsedTime(request?.capturedAt ? dayjs(request?.capturedAt).fromNow() : null),
    1000
  )

  useEffect((): (() => void) => {
    interval.start()

    return interval.stop // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <>
      <Grid>
        {!!request && !!session && showRequestDetails && (
          <>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <Title order={4} mb="md">
                Request details
              </Title>
              <Table my="md" withRowBorders={false} verticalSpacing="0.2em" highlightOnHover>
                <Table.Tbody>
                  <Table.Tr>
                    <Table.Td ta="right" w="15%">
                    Path
                  </Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="80%" />) ||
                      (request.url && <WebHookPath sID={session.sID} url={request.url} />) || <>...</>}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">Method</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="15%" />) || (
                      <Badge color={methodToColor(request.method ?? '')} mb="0.2em">
                        {request.method}
                      </Badge>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">From</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="20%" />) || (
                      <Flex justify="flex-start" align="center">
                        <Text span>{request.clientAddress}</Text>
                        <Flex align="center" ml="md" gap="sm">
                          {[
                            ['WhoIs', 'https://who.is/whois-ip/ip-address/' + request.clientAddress],
                            ['Shodan', 'https://www.shodan.io/host/' + request.clientAddress],
                            ['Netify', 'https://www.netify.ai/resources/ips/' + request.clientAddress],
                            ['Censys', 'https://search.censys.io/hosts/' + request.clientAddress],
                          ].map(([name, link], index) => (
                            <Link key={index} to={link} target="_blank" rel="noreferrer">
                              {name}
                            </Link>
                          ))}
                        </Flex>
                      </Flex>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">When</Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="sm" w="45%" />) || (
                      <>
                        {request.capturedAt && <>{dayjs(request.capturedAt).format('YYYY-MM-DD HH:mm:ss.SSS')}</>}
                        {elapsedTime && <span style={{ paddingLeft: '0.3em' }}>({elapsedTime})</span>}
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
                  <Table.Td ta="right">
                    <Text size="xs" c="dimmed" span>
                      ID
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {(loading && <Skeleton radius="xl" h="xs" w="50%" />) || (
                      <Text size="xs" c="dimmed" span>
                        {request.rID}
                      </Text>
                    )}
                  </Table.Td>
                </Table.Tr>
              </Table.Tbody>
            </Table>
          </Grid.Col>
          <Grid.Col span={{ base: 12, md: 6 }}>
            <Title order={4} mb="md">
              HTTP headers
            </Title>
            {(loading && <Skeleton radius="md" h="10em" w="100%" />) ||
              (!!request.headers && (
                <CodeHighlightTabs
                  code={{
                    fileName: 'headers.txt',
                    code: request.headers.map(({ name, value }) => `${name}: ${value}`).join('\n'),
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
          {!loading && !!request && !!payload && payload.length > 0 && (
            <Button
              variant="light"
              color="indigo"
              size="compact-sm"
              ml="sm"
              leftSection={<IconDownload size="1.2em" />}
              onClick={() => (payload ? download(payload, `${request.rID}.bin`) : undefined)}
            >
              Download
            </Button>
          )}
        </Title>
        {(loading && <Skeleton radius="md" h="8em" w="100%" />) || (
          <Tabs variant="default" defaultValue={TabsList.Text} keepMounted={false}>
            <Tabs.List>
              <Tabs.Tab value={TabsList.Text} leftSection={<IconLetterCase />} color="blue">
                Text
              </Tabs.Tab>
              {!!payload && payload.length > 0 && (
                <Tabs.Tab value={TabsList.Binary} leftSection={<IconBinary />} color="teal">
                  Binary
                </Tabs.Tab>
              )}
            </Tabs.List>
            <Tabs.Panel value={TabsList.Text}>
              <ViewText input={payload || null} contentType={contentType} />
            </Tabs.Panel>
            {!!payload && payload.length > 0 && (
              <Tabs.Panel value={TabsList.Binary}>
                <ViewHex input={payload} />
              </Tabs.Panel>
            )}
          </Tabs>
        )}
      </Grid.Col>
      {!!request && !!request.forwardedRequests && request.forwardedRequests.length > 0 && showRequestDetails && (
        <Grid.Col span={12}>
          <Title order={4} my="md">
            Forwarded Requests ({request.forwardedRequests.length})
          </Title>
          <Accordion variant="separated" defaultValue={request.forwardedRequests[0]?.url}>
            {request.forwardedRequests.map((fr, index) => (
              <Accordion.Item key={index} value={`${fr.url}-${index}`}>
                <Accordion.Control>
                  <Group justify="space-between">
                    <Text size="sm" style={{ flexGrow: 1 }}>
                      <Text span fw={500}>URL:</Text> {fr.url}
                    </Text>
                    {fr.error ? (
                      <Badge color="red">Error</Badge>
                    ) : (
                      <Badge color={fr.statusCode && fr.statusCode >= 200 && fr.statusCode < 300 ? 'teal' : 'orange'}>
                        Status: {fr.statusCode || 'N/A'}
                      </Badge>
                    )}
                    <Text size="xs" c="dimmed">
                      Attempted: {dayjs(fr.occurredAt).format('YYYY-MM-DD HH:mm:ss.SSS')}
                    </Text>
                  </Group>
                </Accordion.Control>
                <Accordion.Panel>
                  {fr.error && (
                    <Paper p="md" withBorder shadow="xs" mb="sm">
                      <Text c="red" fw={500}>Error during forwarding:</Text>
                      <Text>{fr.error}</Text>
                    </Paper>
                  )}
                  <Tabs defaultValue="requestToProxy" variant="outline">
                    <Tabs.List>
                      <Tabs.Tab value="requestToProxy">Request to Proxy</Tabs.Tab>
                      <Tabs.Tab value="responseFromProxy" disabled={!!fr.error}>Response from Proxy</Tabs.Tab>
                    </Tabs.List>

                    <Tabs.Panel value="requestToProxy" pt="xs">
                      {fr.requestHeaders && fr.requestHeaders.length > 0 && (
                        <>
                          <Text fw={500} mb="xs">Request Headers:</Text>
                          <CodeHighlightTabs
                            code={{
                              fileName: 'headers.txt',
                              code: fr.requestHeaders.map(({ name, value }) => `${name}: ${value}`).join('\n'),
                              language: 'bash',
                            }}
                            maxCollapsedHeight="10em"
                            withExpandButton
                            withCopyButton
                          />
                        </>
                      )}
                      {fr.requestBody && fr.requestBody.length > 0 && (
                        <>
                          <Text fw={500} mt="sm" mb="xs">Request Body:</Text>
                          <ViewText input={fr.requestBody} contentType="application/octet-stream" /> {/* Adjust content type as needed */}
                        </>
                      )}
                      {(!fr.requestHeaders || fr.requestHeaders.length === 0) && (!fr.requestBody || fr.requestBody.length === 0) && (
                        <Text c="dimmed">No request headers or body sent to proxy.</Text>
                      )}
                    </Tabs.Panel>

                    <Tabs.Panel value="responseFromProxy" pt="xs">
                      {!fr.error && (
                        <>
                          {fr.responseHeaders && fr.responseHeaders.length > 0 && (
                            <>
                              <Text fw={500} mb="xs">Response Headers:</Text>
                              <CodeHighlightTabs
                                code={{
                                  fileName: 'headers.txt',
                                  code: fr.responseHeaders.map(({ name, value }) => `${name}: ${value}`).join('\n'),
                                  language: 'bash',
                                }}
                                maxCollapsedHeight="10em"
                                withExpandButton
                                withCopyButton
                              />
                            </>
                          )}
                           {fr.responseBody && fr.responseBody.length > 0 && (
                            <>
                              <Text fw={500} mt="sm" mb="xs">Response Body:</Text>
                              <ViewText input={fr.responseBody} contentType={fr.responseHeaders?.find(h => h.name.toLowerCase() === 'content-type')?.value || 'application/octet-stream'} />
                            </>
                          )}
                          {(!fr.responseHeaders || fr.responseHeaders.length === 0) && (!fr.responseBody || fr.responseBody.length === 0) && (
                            <Text c="dimmed">No response headers or body received from proxy.</Text>
                          )}
                        </>
                      )}
                    </Tabs.Panel>
                  </Tabs>
                </Accordion.Panel>
              </Accordion.Item>
            ))}
          </Accordion>
        </Grid.Col>
      )}
    </Grid>
    </>
  )
}

enum TabsList {
  Text = 'Text',
  Binary = 'Binary',
}
export const WebHookPath: React.FC<{ sID: string; url: URL }> = ({ sID, url }) => {
  const { search, hash } = url // search may be '', '?' or '?key=value'; hash may be '', '#' or '#fragment'
  let { pathname } = url // pathname is usually '/{sID}' or '/{sID}/any/path'

  // remove the sID from the pathname since it's already displayed and useless a bit
  if (pathname.startsWith('/' + sID)) {
    pathname = pathname.slice(sID.length + 1)
  }

  // if the pathname is empty, set it to '/'
  if (pathname === '') {
    pathname = '/'
  }

  return (
    <Text size="md" style={{ wordBreak: 'break-all' }}>
      <Text span>{pathname}</Text>
      {search && (
        <Text variant="gradient" gradient={{ from: 'yellow', to: 'orange', deg: 90 }} span>
          {search}
        </Text>
      )}
      {hash && <Text c="dimmed">{hash}</Text>}
      <Button
        variant="light"
        color="gray"
        size="compact-xs"
        component="a"
        href={`${url.pathname}${search}${hash}`}
        target="_blank"
        ml="sm"
        mb="0.1em"
      >
        Open
      </Button>
    </Text>
  )
}

const download = (data: Readonly<Uint8Array>, fileName: string): void => {
  const blob = new Blob([data.buffer], { type: 'application/octet-stream' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')

  a.href = url
  a.download = fileName
  a.click()

  setTimeout(() => {
    URL.revokeObjectURL(url)

    a.remove()
  }, 1000) // 1s
}

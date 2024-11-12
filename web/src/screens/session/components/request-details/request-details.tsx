import React, { useEffect, useState } from 'react'
import { CodeHighlightTabs } from '@mantine/code-highlight'
import { Badge, Button, Flex, Grid, Skeleton, Table, Tabs, Text, Title } from '@mantine/core'
import { useInterval } from '@mantine/hooks'
import { Link } from 'react-router-dom'
import { IconBinary, IconDownload, IconLetterCase } from '@tabler/icons-react'
import dayjs from 'dayjs'
import { useData, UsedStorageKeys, useSettings, useStorage } from '~/shared'
import { methodToColor } from '~/theme'
import { ViewHex, ViewText } from './components'

export const RequestDetails = (): React.JSX.Element => {
  const { session, request, requestLoading } = useData()
  const { showRequestDetails } = useSettings()

  const [headersExpanded, setHeadersExpanded] = useStorage<boolean>(false, UsedStorageKeys.RequestDetailsHeadersExpand)
  const [elapsedTime, setElapsedTime] = useState<string | null>(null)
  const [contentType, setContentType] = useState<string | null>(null)

  useEffect(
    () => setContentType(request?.headers.find(({ name }) => name.toLowerCase() === 'content-type')?.value ?? null),
    [request]
  )

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
                    {(requestLoading && <Skeleton radius="xl" h="sm" w="80%" />) ||
                      (request.url && <WebHookPath sID={session.sID} url={request.url} />) || <>...</>}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">Method</Table.Td>
                  <Table.Td>
                    {(requestLoading && <Skeleton radius="xl" h="sm" w="15%" />) || (
                      <Badge color={methodToColor(request.method ?? '')} mb="0.2em">
                        {request.method}
                      </Badge>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">From</Table.Td>
                  <Table.Td>
                    {(requestLoading && <Skeleton radius="xl" h="sm" w="20%" />) || (
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
                    {(requestLoading && <Skeleton radius="xl" h="sm" w="45%" />) || (
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
                    {(requestLoading && <Skeleton radius="xl" h="sm" w="15%" />) || (
                      <>{request.payload?.length} bytes</>
                    )}
                  </Table.Td>
                </Table.Tr>
                <Table.Tr>
                  <Table.Td ta="right">
                    <Text size="xs" c="dimmed" span>
                      ID
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {(requestLoading && <Skeleton radius="xl" h="xs" w="50%" />) || (
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
            {(requestLoading && <Skeleton radius="md" h="10em" w="100%" />) ||
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
          {!requestLoading && !!request?.payload && request.payload.length > 0 && (
            <Button
              variant="light"
              color="indigo"
              size="compact-sm"
              ml="sm"
              leftSection={<IconDownload size="1.2em" />}
              onClick={() => (request.payload ? download(request.payload, `${request.rID}.bin`) : undefined)}
            >
              Download
            </Button>
          )}
        </Title>
        {(requestLoading && <Skeleton radius="md" h="8em" w="100%" />) || (
          <Tabs variant="default" defaultValue={TabsList.Text} keepMounted={false}>
            <Tabs.List>
              <Tabs.Tab value={TabsList.Text} leftSection={<IconLetterCase />} color="blue">
                Text
              </Tabs.Tab>
              {!!request?.payload && request.payload.length > 0 && (
                <Tabs.Tab value={TabsList.Binary} leftSection={<IconBinary />} color="teal">
                  Binary
                </Tabs.Tab>
              )}
            </Tabs.List>
            <Tabs.Panel value={TabsList.Text}>
              <ViewText input={request?.payload || null} contentType={contentType} />
            </Tabs.Panel>
            {!!request?.payload && request.payload.length > 0 && (
              <Tabs.Panel value={TabsList.Binary}>
                <ViewHex input={request.payload} />
              </Tabs.Panel>
            )}
          </Tabs>
        )}
      </Grid.Col>
    </Grid>
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

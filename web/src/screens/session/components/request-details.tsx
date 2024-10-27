import { Badge, Code, Grid, Skeleton, Table } from '@mantine/core'
import React, { useEffect } from 'react'
import type { Client } from '~/api'

export default function RequestDetails({
  apiClient,
  sID,
  rID,
}: {
  apiClient: Client
  sID: string
  rID: string
}): React.JSX.Element {
  const [loading, setLoading] = React.useState<boolean>(true)
  const [url, setUrl] = React.useState<URL | null>(null)
  const [method, setMethod] = React.useState<string | null>(null)
  const [clientAddress, setClientAddress] = React.useState<string | null>(null)
  const [capturedAt, setCapturedAt] = React.useState<Date | null>(null)
  const [payload, setPayload] = React.useState<Uint8Array | null>(null)
  const [headers, setHeaders] = React.useState<ReadonlyArray<{ name: string; value: string }> | null>(null)

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

  return (
    <Grid>
      <Grid.Col span={{ base: 12, md: 6 }}>
        <Table my="md" withRowBorders={false} highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th w="15%" />
              <Table.Th>Request details</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            <Table.Tr>
              <Table.Td ta="right">URL</Table.Td>
              <Table.Td>{(loading && <Skeleton radius="xl" h="sm" w="80%" />) || <>{url?.toString()}</>}</Table.Td>
            </Table.Tr>
            <Table.Tr>
              <Table.Td ta="right">Method</Table.Td>
              <Table.Td>{(loading && <Skeleton radius="xl" h="sm" w="15%" />) || <Badge>{method}</Badge>}</Table.Td>
            </Table.Tr>
            <Table.Tr>
              <Table.Td ta="right">From</Table.Td>
              <Table.Td>{(loading && <Skeleton radius="xl" h="sm" w="20%" />) || <>{clientAddress}</>}</Table.Td>
            </Table.Tr>
            <Table.Tr>
              <Table.Td ta="right">When</Table.Td>
              <Table.Td>
                {(loading && <Skeleton radius="xl" h="sm" w="45%" />) || (
                  <>{capturedAt?.toString()} (TODO: add elapsed time)</>
                )}
              </Table.Td>
            </Table.Tr>
            <Table.Tr>
              <Table.Td ta="right">Size</Table.Td>
              <Table.Td>{(loading && <Skeleton radius="xl" h="sm" w="15%" />) || <>31 bytes</>}</Table.Td>
            </Table.Tr>
            <Table.Tr>
              <Table.Td ta="right">ID</Table.Td>
              <Table.Td>
                {(loading && <Skeleton radius="xl" h="sm" w="50%" />) || <>{payload?.length} bytes</>}
              </Table.Td>
            </Table.Tr>
          </Table.Tbody>
        </Table>
      </Grid.Col>

      <Grid.Col span={{ base: 12, md: 6 }}>
        <Table my="md" withRowBorders={false} highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th w="28%" />
              <Table.Th>HTTP headers</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {(loading &&
              [...Array(4)].map((_, i) => (
                <Table.Tr key={i}>
                  <Table.Td ta="right">
                    <Skeleton radius="xl" h="sm" w="80%" />
                  </Table.Td>
                  <Table.Td>
                    <Skeleton radius="xl" h="sm" w="80%" />
                  </Table.Td>
                </Table.Tr>
              ))) ||
              headers?.map(({ name, value }) => (
                <Table.Tr key={name}>
                  <Table.Td ta="right" style={{ fontFamily: 'monospace', fontSize: '0.9em' }}>
                    {name}
                  </Table.Td>
                  <Table.Td>
                    <Code>{value}</Code>
                  </Table.Td>
                </Table.Tr>
              ))}
          </Table.Tbody>
        </Table>
      </Grid.Col>
    </Grid>
  )
}

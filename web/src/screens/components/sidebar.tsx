import {
  Center,
  Loader,
  Stack,
  Text,
  UnstyledButton,
  Badge,
  CloseButton,
  Flex,
  Title,
  type MantineColor,
  Button,
} from '@mantine/core'
import { useInterval } from '@mantine/hooks'
import { IconTrash } from '@tabler/icons-react'
import dayjs from 'dayjs'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import styles from './sidebar.module.css'

export type ListedRequest = {
  id: string
  method: string
  clientAddress: string
  capturedAt: Date
}

const Request = ({
  sID,
  request,
  isActive = false,
  onDelete,
}: {
  sID: string
  request: ListedRequest
  isActive?: boolean
  onDelete?: (sID: string, rID: string) => void
}): React.JSX.Element => {
  const formatDateTime = (date: Date): string => dayjs(date).fromNow()
  const [elapsedTime, setElapsedTime] = useState<string>(formatDateTime(request.capturedAt))
  const interval = useInterval(() => setElapsedTime(formatDateTime(request.capturedAt)), 1000)

  useEffect((): (() => void) => {
    interval.start()

    return interval.stop // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <Badge
      variant={isActive ? 'gradient' : 'default'}
      gradient={isActive ? { from: 'teal', to: 'cyan', deg: 90 } : undefined}
      radius="sm"
      h="auto"
      w="100%"
      p={0}
      className={styles.requestBlock}
      styles={{
        root: {
          textTransform: 'none',
          borderStyle: 'solid',
          borderWidth: '0.1em',
          borderColor: isActive ? 'transparent' : undefined,
        },
        label: { width: '100%' },
      }}
      fullWidth
    >
      <Flex justify="space-between">
        <UnstyledButton
          component={Link}
          p="sm"
          pr={0}
          style={{ width: '100%' }}
          to={pathTo(RouteIDs.SessionAndRequest, sID, request.id)}
        >
          <Flex align="center">
            <Title order={4} style={{ fontWeight: 500 }}>
              {request.clientAddress}
            </Title>
            <Badge
              variant="dot"
              ml="xs"
              styles={{ label: { fontWeight: 300, cursor: 'pointer' } }}
              color={methodColor(request.method)}
            >
              {request.method}
            </Badge>
          </Flex>
          <Text size="sm">
            {dayjs(request.capturedAt).format('h:mm:ss a')}
            <Text size="xs" pl="0.5em" span>
              ({elapsedTime})
            </Text>
          </Text>
        </UnstyledButton>
        {!!onDelete && (
          <CloseButton
            size={16}
            iconSize={16}
            m="sm"
            ml={0}
            aria-label="Delete"
            title="Delete"
            onClick={() => onDelete(sID, request.id)}
          />
        )}
      </Flex>
    </Badge>
  )
}

const methodColor = (method: string): MantineColor => {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'blue'
    case 'POST':
      return 'green'
    case 'PUT':
      return 'yellow'
    case 'DELETE':
      return 'red'
    case 'PATCH':
      return 'purple'
    case 'HEAD':
      return 'teal'
    case 'OPTIONS':
      return 'orange'
    case 'TRACE':
      return 'pink'
    case 'CONNECT':
      return 'indigo'
  }

  return 'gray'
}

const NoRequests = (): React.JSX.Element => (
  <Center pt="2em">
    <Loader color="dimmed" size="1em" mr={8} mb={3} />
    <Text c="dimmed">Waiting for first request</Text>
  </Center>
)

const NoSession = (): React.JSX.Element => (
  <Center pt="2em">
    <Loader color="dimmed" size="1em" mr={8} mb={3} />
    <Text c="dimmed">No session selected</Text>
  </Center>
)

export default function SideBar({
  sID,
  rID,
  requests,
  onRequestDelete,
  onAllRequestsDelete,
}: {
  sID: string | null
  rID: string | null
  requests: ReadonlyArray<ListedRequest>
  onRequestDelete?: (sID: string, rID: string) => void
  onAllRequestsDelete?: (sID: string) => void
}): React.JSX.Element {
  return (
    <Stack align="stretch" justify="flex-start" gap="xs">
      {(!!sID &&
        ((!!requests.length && (
          <>
            {requests.map((request) => (
              <Request
                sID={sID}
                request={request}
                key={request.id}
                isActive={rID === request.id}
                onDelete={onRequestDelete && (() => onRequestDelete(sID, request.id))}
              />
            ))}
            {requests.length > 1 && !!onAllRequestsDelete && (
              <Center>
                <Button
                  leftSection={<IconTrash size="1em" />}
                  size="compact-xs"
                  variant="outline"
                  color="red"
                  px="xs"
                  mb="sm"
                  radius="xl"
                  opacity={0.7}
                  onClick={() => onAllRequestsDelete(sID)}
                >
                  Delete all requests
                </Button>
              </Center>
            )}
          </>
        )) || <NoRequests />)) || <NoSession />}
    </Stack>
  )
}

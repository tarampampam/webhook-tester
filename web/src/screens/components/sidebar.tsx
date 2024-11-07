import {
  Badge,
  Button,
  type ButtonProps,
  Center,
  CloseButton,
  Flex,
  Group,
  Image,
  Loader,
  Stack,
  Text,
  UnstyledButton,
} from '@mantine/core'
import { useInterval } from '@mantine/hooks'
import { IconChevronDown, IconChevronsDown, IconChevronsUp, IconChevronUp, IconTrash } from '@tabler/icons-react'
import dayjs from 'dayjs'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import { methodToColor } from '~/theme'
import PandaSvg from '~/assets/panda.svg'
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
            <Text size="xl" fw={500} style={{ flex: 1, width: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {request.clientAddress}
            </Text>
            <Badge
              variant="dot"
              mx="xs"
              styles={{ label: { fontWeight: 300, cursor: 'pointer' } }}
              color={methodToColor(request.method)}
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

const Navigator = ({
  sID,
  rID,
  requests,
}: {
  sID: string
  rID: string | null
  requests: ReadonlyArray<ListedRequest>
}) => {
  const [jumpFirstEnabled, setJumpFirstEnabled] = useState<boolean>(false)
  const [jumpPrevEnabled, setJumpPrevEnabled] = useState<boolean>(false)
  const [jumpNextEnabled, setJumpNextEnabled] = useState<boolean>(false)
  const [jumpLastEnabled, setJumpLastEnabled] = useState<boolean>(false)

  const [pathToFirst, setPathToFirst] = useState<string | null>(null)
  const [pathToPrev, setPathToPrev] = useState<string | null>(null)
  const [pathToNext, setPathToNext] = useState<string | null>(null)
  const [pathToLast, setPathToLast] = useState<string | null>(null)

  useEffect(() => {
    const firstIdx = 0
    const prevIdx = requests.findIndex((req) => req.id === rID) + 1
    const nextIdx = requests.findIndex((req) => req.id === rID) - 1
    const lastIdx = requests.length - 1

    const firstID = requests[firstIdx] ? requests[firstIdx].id : null
    const prevID = requests[prevIdx] ? requests[prevIdx].id : null
    const nextID = requests[nextIdx] ? requests[nextIdx].id : null
    const lastID = requests[lastIdx] ? requests[lastIdx].id : null
    const moreThanOneRequest = requests.length > 1

    setJumpFirstEnabled(moreThanOneRequest && firstID !== rID)
    setJumpPrevEnabled(moreThanOneRequest && !!prevID && !!rID && rID !== lastID)
    setJumpNextEnabled(moreThanOneRequest && !!nextID && !!rID && rID !== firstID)
    setJumpLastEnabled(moreThanOneRequest && lastID !== rID)

    setPathToFirst(moreThanOneRequest && firstID ? pathTo(RouteIDs.SessionAndRequest, sID, firstID) : null)
    setPathToPrev(moreThanOneRequest && prevID && rID ? pathTo(RouteIDs.SessionAndRequest, sID, prevID) : null)
    setPathToNext(moreThanOneRequest && nextID && rID ? pathTo(RouteIDs.SessionAndRequest, sID, nextID) : null)
    setPathToLast(moreThanOneRequest && lastID ? pathTo(RouteIDs.SessionAndRequest, sID, lastID) : null)
  }, [requests, sID, rID])

  const shortJumpButtonProps: Partial<ButtonProps> = {
    variant: 'default',
    size: 'compact-xs',
  }

  const longJumpButtonProps: Partial<ButtonProps> = {
    ...shortJumpButtonProps,
    styles: { section: { margin: 0 } },
  }

  return (
    <Group justify="space-between">
      <Button.Group>
        <Button // jump to the first request
          {...longJumpButtonProps}
          leftSection={<IconChevronsUp size="1em" />}
          disabled={!jumpFirstEnabled}
          renderRoot={(props) =>
            jumpFirstEnabled && pathToFirst ? <Link to={pathToFirst} {...props} /> : <button {...props} />
          }
          title="First request"
        />
        <Button // jump to the next request
          {...shortJumpButtonProps}
          leftSection={<IconChevronUp size="1em" />}
          disabled={!jumpNextEnabled}
          renderRoot={(props) =>
            jumpNextEnabled && pathToNext ? <Link to={pathToNext} {...props} /> : <button {...props} />
          }
        >
          Newer
        </Button>
      </Button.Group>

      <Button.Group>
        <Button // jump to the previous request
          {...shortJumpButtonProps}
          rightSection={<IconChevronDown size="1em" />}
          disabled={!jumpPrevEnabled}
          renderRoot={(props) =>
            jumpPrevEnabled && pathToPrev ? <Link to={pathToPrev} {...props} /> : <button {...props} />
          }
        >
          Older
        </Button>
        <Button // jump to the last request
          {...longJumpButtonProps}
          leftSection={<IconChevronsDown size="1em" />}
          disabled={!jumpLastEnabled}
          renderRoot={(props) =>
            jumpLastEnabled && pathToLast ? <Link to={pathToLast} {...props} /> : <button {...props} />
          }
          title="Last request"
        />
      </Button.Group>
    </Group>
  )
}

const NoRequests = (): React.JSX.Element => (
  <Stack gap="xs" h="100%" justify="space-between">
    <Center pt="2em">
      <Image src={PandaSvg} w="50%" />
    </Center>
    <Center>
      <Loader color="dimmed" size="1em" mr={8} mb={3} />
      <Text c="dimmed">Waiting for first request</Text>
    </Center>
  </Stack>
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
            <Navigator sID={sID} rID={rID} requests={requests} />

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

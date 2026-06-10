import React, { useCallback, useEffect, useMemo, useState, type ForwardedRef } from 'react'
import { notifications as notify } from '@mantine/notifications'
import { Badge, CloseButton, Flex, Text, UnstyledButton } from '@mantine/core'
import { useInterval } from '@mantine/hooks'
import dayjs from 'dayjs'
import { Link, useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID } from '~/routing'
import { type Request as RequestData, useRequests } from '~/shared'
import { methodToColor } from '~/theme'
import styles from './request.module.css'

type TinyRequest = Omit<RequestData, 'getPayload'>

export const Request = ({
  sID,
  request,
  isActive = false,
  componentRef,
}: {
  sID: string
  request: TinyRequest
  isActive?: boolean
  componentRef?: ForwardedRef<HTMLDivElement>
}): React.JSX.Element => {
  const navigate = useNavigate()
  const { delRequest, requests: requestsMap } = useRequests()

  const requests = useMemo(
    () => [...requestsMap.values()].sort((a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()),
    [requestsMap]
  )

  const formatDateTime = (date: Date): string => dayjs(date).fromNow()
  const [elapsedTime, setElapsedTime] = useState<string>(formatDateTime(request.capturedAt))
  const interval = useInterval(() => setElapsedTime(formatDateTime(request.capturedAt)), 1000)

  /** Delete request */
  const handleDelete = useCallback(() => {
    const requestIdx: number | -1 = requests.findIndex((r) => r.id === request.id)
    const [nextRequest, prevRequest]: [TinyRequest | undefined, TinyRequest | undefined] = [
      requestIdx !== -1 ? requests[requestIdx + 1] : undefined,
      requestIdx !== -1 ? requests[requestIdx - 1] : undefined,
    ]

    delRequest(sID, request.id).catch((err) => {
      notify.show({ title: 'Failed to delete request', message: String(err), color: 'red', autoClose: 5000 })
    })

    // if the request is currently opened, navigate to the next one
    if (nextRequest) {
      navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID, rID: nextRequest.id }))
    } else if (prevRequest) {
      // if there is no next request, navigate to the previous one
      navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID, rID: prevRequest.id }))
    } else {
      // if there is no next request, navigate to the session
      navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID }))
    }
  }, [sID, request.id, navigate, delRequest, requests])

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
      ref={componentRef}
      fullWidth
    >
      <Flex justify="space-between">
        <UnstyledButton
          component={Link}
          p="sm"
          pr={0}
          style={{ width: '100%' }}
          to={pathTo(ROUTE_ID.SessionAndRequest, { sID, rID: request.id })}
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
        <CloseButton size={16} iconSize={16} m="sm" ml={0} aria-label="Delete" title="Delete" onClick={handleDelete} />
      </Flex>
    </Badge>
  )
}

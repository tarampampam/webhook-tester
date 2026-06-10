import React, { useEffect, useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button, type ButtonProps, Group, Badge } from '@mantine/core'
import { IconChevronDown, IconChevronsDown, IconChevronsUp, IconChevronUp } from '@tabler/icons-react'
import { pathTo, ROUTE_ID, useActiveRequestID, useActiveSessionID } from '~/routing'
import { useRequests } from '~/shared'

export const Navigator = (): React.JSX.Element => {
  const { requests: requestsMap } = useRequests()
  const activeSessionID = useActiveSessionID()
  const activeRequestID = useActiveRequestID()
  const navigate = useNavigate()

  const requests = useMemo(
    () => [...requestsMap.values()].sort((a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()),
    [requestsMap]
  )

  const navInfo = useMemo(() => {
    const firstIdx: number = 0
    const prevIdx: number | -1 = requests.findIndex((rq) => !!activeRequestID && rq.id === activeRequestID) + 1
    const nextIdx: number | -1 = requests.findIndex((rq) => !!activeRequestID && rq.id === activeRequestID) - 1
    const lastIdx: number | -1 = requests.length - 1

    const firstID = requests[firstIdx] ? requests[firstIdx].id : null
    const prevID = requests[prevIdx] ? requests[prevIdx].id : null
    const nextID = requests[nextIdx] ? requests[nextIdx].id : null
    const lastID = requests[lastIdx] ? requests[lastIdx].id : null
    const moreThanOneRequest = requests.length > 1

    return {
      jumpFirstEnabled: moreThanOneRequest && !!activeRequestID && firstID !== activeRequestID,
      jumpPrevEnabled: moreThanOneRequest && !!activeRequestID && !!prevID && activeRequestID !== lastID,
      jumpNextEnabled: moreThanOneRequest && !!activeRequestID && !!nextID && activeRequestID !== firstID,
      jumpLastEnabled: moreThanOneRequest && !!activeRequestID && lastID !== activeRequestID,

      pathToFirst:
        moreThanOneRequest && !!activeSessionID && firstID
          ? pathTo(ROUTE_ID.SessionAndRequest, { sID: activeSessionID, rID: firstID })
          : null,
      pathToPrev:
        moreThanOneRequest && !!activeSessionID && prevID && !!activeRequestID
          ? pathTo(ROUTE_ID.SessionAndRequest, { sID: activeSessionID, rID: prevID })
          : null,
      pathToNext:
        moreThanOneRequest && !!activeSessionID && nextID && !!activeRequestID
          ? pathTo(ROUTE_ID.SessionAndRequest, { sID: activeSessionID, rID: nextID })
          : null,
      pathToLast:
        moreThanOneRequest && !!activeSessionID && lastID
          ? pathTo(ROUTE_ID.SessionAndRequest, { sID: activeSessionID, rID: lastID })
          : null,
    }
  }, [activeRequestID, activeSessionID, requests])

  // listen for arrow keys to navigate between requests
  useEffect(() => {
    const eventsHandler = (e: KeyboardEvent) => {
      if ((e.code === 'ArrowDown' || e.code === 'ArrowRight') && navInfo.jumpPrevEnabled && navInfo.pathToPrev) {
        navigate(navInfo.pathToPrev)
      } else if ((e.code === 'ArrowUp' || e.code === 'ArrowLeft') && navInfo.jumpNextEnabled && navInfo.pathToNext) {
        navigate(navInfo.pathToNext)
      }
    }

    window.addEventListener('keydown', eventsHandler)

    return () => window.removeEventListener('keydown', eventsHandler)
  }, [navInfo, navigate])

  const shortJumpButtonProps: Partial<ButtonProps> = { variant: 'default', size: 'compact-xs' }
  const longJumpButtonProps: Partial<ButtonProps> = { ...shortJumpButtonProps, styles: { section: { margin: 0 } } }

  return (
    <Group justify="space-between">
      <Button.Group>
        <Button // jump to the first request
          {...longJumpButtonProps}
          leftSection={<IconChevronsUp size="1em" />}
          disabled={!navInfo.jumpFirstEnabled}
          renderRoot={(props) =>
            navInfo.jumpFirstEnabled && navInfo.pathToFirst ? (
              <Link to={navInfo.pathToFirst} {...props} />
            ) : (
              <button {...props} />
            )
          }
          title="First request"
        />
        <Button // jump to the next request
          {...shortJumpButtonProps}
          leftSection={<IconChevronUp size="1em" />}
          disabled={!navInfo.jumpNextEnabled}
          renderRoot={(props) =>
            navInfo.jumpNextEnabled && navInfo.pathToNext ? (
              <Link to={navInfo.pathToNext} {...props} />
            ) : (
              <button {...props} />
            )
          }
        >
          Newer
        </Button>
      </Button.Group>

      {requests.length && (
        <Badge color="gray" size="xs" px="xs">
          {requests.length}
        </Badge>
      )}

      <Button.Group>
        <Button // jump to the previous request
          {...shortJumpButtonProps}
          rightSection={<IconChevronDown size="1em" />}
          disabled={!navInfo.jumpPrevEnabled}
          renderRoot={(props) =>
            navInfo.jumpPrevEnabled && navInfo.pathToPrev ? (
              <Link to={navInfo.pathToPrev} {...props} />
            ) : (
              <button {...props} />
            )
          }
        >
          Older
        </Button>
        <Button // jump to the last request
          {...longJumpButtonProps}
          leftSection={<IconChevronsDown size="1em" />}
          disabled={!navInfo.jumpLastEnabled}
          renderRoot={(props) =>
            navInfo.jumpLastEnabled && navInfo.pathToLast ? (
              <Link to={navInfo.pathToLast} {...props} />
            ) : (
              <button {...props} />
            )
          }
          title="Last request"
        />
      </Button.Group>
    </Group>
  )
}

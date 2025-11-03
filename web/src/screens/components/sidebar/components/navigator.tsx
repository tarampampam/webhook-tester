import React, { useEffect, useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button, type ButtonProps, Group, Badge } from '@mantine/core'
import { IconChevronDown, IconChevronsDown, IconChevronsUp, IconChevronUp } from '@tabler/icons-react'
import { pathTo, RouteIDs } from '~/routing'
import { useData } from '~/shared'

export const Navigator = (): React.JSX.Element => {
  const { session, request, requests } = useData()
  const navigate = useNavigate()

  const navInfo = useMemo(() => {
    const firstIdx: number = 0
    const prevIdx: number | -1 = requests.findIndex((rq) => !!request && rq.rID === request.rID) + 1
    const nextIdx: number | -1 = requests.findIndex((rq) => !!request && rq.rID === request.rID) - 1
    const lastIdx: number | -1 = requests.length - 1

    const firstID = requests[firstIdx] ? requests[firstIdx].rID : null
    const prevID = requests[prevIdx] ? requests[prevIdx].rID : null
    const nextID = requests[nextIdx] ? requests[nextIdx].rID : null
    const lastID = requests[lastIdx] ? requests[lastIdx].rID : null
    const moreThanOneRequest = requests.length > 1

    return {
      jumpFirstEnabled: moreThanOneRequest && !!request && firstID !== request.rID,
      jumpPrevEnabled: moreThanOneRequest && !!request && !!prevID && request.rID !== lastID,
      jumpNextEnabled: moreThanOneRequest && !!request && !!nextID && request.rID !== firstID,
      jumpLastEnabled: moreThanOneRequest && !!request && lastID !== request.rID,

      pathToFirst:
        moreThanOneRequest && !!session && firstID ? pathTo(RouteIDs.SessionAndRequest, session.sID, firstID) : null,
      pathToPrev:
        moreThanOneRequest && !!session && prevID && !!request
          ? pathTo(RouteIDs.SessionAndRequest, session.sID, prevID)
          : null,
      pathToNext:
        moreThanOneRequest && !!session && nextID && !!request
          ? pathTo(RouteIDs.SessionAndRequest, session.sID, nextID)
          : null,
      pathToLast:
        moreThanOneRequest && !!session && lastID ? pathTo(RouteIDs.SessionAndRequest, session.sID, lastID) : null,
    }
  }, [request, requests, session])

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

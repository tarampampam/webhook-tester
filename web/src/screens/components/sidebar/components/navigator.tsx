import React, { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button, type ButtonProps, Group, Badge } from '@mantine/core'
import { IconChevronDown, IconChevronsDown, IconChevronsUp, IconChevronUp } from '@tabler/icons-react'
import { pathTo, RouteIDs } from '~/routing'
import { useData } from '~/shared'

export const Navigator = (): React.JSX.Element => {
  const { session, request, requests } = useData()
  const navigate = useNavigate()

  const [jumpFirstEnabled, setJumpFirstEnabled] = useState<boolean>(false)
  const [jumpPrevEnabled, setJumpPrevEnabled] = useState<boolean>(false)
  const [jumpNextEnabled, setJumpNextEnabled] = useState<boolean>(false)
  const [jumpLastEnabled, setJumpLastEnabled] = useState<boolean>(false)

  const [pathToFirst, setPathToFirst] = useState<string | null>(null)
  const [pathToPrev, setPathToPrev] = useState<string | null>(null)
  const [pathToNext, setPathToNext] = useState<string | null>(null)
  const [pathToLast, setPathToLast] = useState<string | null>(null)

  useEffect(() => {
    const firstIdx: number = 0
    const prevIdx: number | -1 = requests.findIndex((rq) => !!request && rq.rID === request.rID) + 1
    const nextIdx: number | -1 = requests.findIndex((rq) => !!request && rq.rID === request.rID) - 1
    const lastIdx: number | -1 = requests.length - 1

    const firstID = requests[firstIdx] ? requests[firstIdx].rID : null
    const prevID = requests[prevIdx] ? requests[prevIdx].rID : null
    const nextID = requests[nextIdx] ? requests[nextIdx].rID : null
    const lastID = requests[lastIdx] ? requests[lastIdx].rID : null
    const moreThanOneRequest = requests.length > 1

    setJumpFirstEnabled(moreThanOneRequest && !!request && firstID !== request.rID)
    setJumpPrevEnabled(moreThanOneRequest && !!request && !!prevID && request.rID !== lastID)
    setJumpNextEnabled(moreThanOneRequest && !!request && !!nextID && request.rID !== firstID)
    setJumpLastEnabled(moreThanOneRequest && !!request && lastID !== request.rID)

    setPathToFirst(
      moreThanOneRequest && !!session && firstID ? pathTo(RouteIDs.SessionAndRequest, session.sID, firstID) : null
    )
    setPathToPrev(
      moreThanOneRequest && !!session && prevID && !!request
        ? pathTo(RouteIDs.SessionAndRequest, session.sID, prevID)
        : null
    )
    setPathToNext(
      moreThanOneRequest && !!session && nextID && !!request
        ? pathTo(RouteIDs.SessionAndRequest, session.sID, nextID)
        : null
    )
    setPathToLast(
      moreThanOneRequest && !!session && lastID ? pathTo(RouteIDs.SessionAndRequest, session.sID, lastID) : null
    )
  }, [request, requests, session])

  // listen for arrow keys to navigate between requests
  useEffect(() => {
    const eventsHandler = (e: KeyboardEvent) => {
      if ((e.code === 'ArrowDown' || e.code === 'ArrowRight') && jumpPrevEnabled && pathToPrev) {
        navigate(pathToPrev)
      } else if ((e.code === 'ArrowUp' || e.code === 'ArrowLeft') && jumpNextEnabled && pathToNext) {
        navigate(pathToNext)
      }
    }

    window.addEventListener('keydown', eventsHandler)

    return () => window.removeEventListener('keydown', eventsHandler)
  })

  const shortJumpButtonProps: Partial<ButtonProps> = { variant: 'default', size: 'compact-xs' }
  const longJumpButtonProps: Partial<ButtonProps> = { ...shortJumpButtonProps, styles: { section: { margin: 0 } } }

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

      {requests.length && (
        <Badge color="gray" size="xs" px="xs">
          {requests.length}
        </Badge>
      )}

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

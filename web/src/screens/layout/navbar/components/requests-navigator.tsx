import type { ButtonProps, GroupProps, SkeletonProps } from '@mantine/core'
import { Badge, Button, Group, Skeleton, useComputedColorScheme } from '@mantine/core'
import { IconChevronDown, IconChevronsDown, IconChevronsUp, IconChevronUp } from '@tabler/icons-react'
import React, { useEffect, useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveRequestID, useActiveSessionID } from '~/routing'
import { L10nKey, useL10n, useRequests } from '~/shared'

export const RequestsNavigator = Object.assign(
  ({ ...props }: GroupProps): React.JSX.Element => {
    const { requests } = useRequests()
    const [sID, rID] = [useActiveSessionID(), useActiveRequestID()]
    const navigate = useNavigate()
    const { t } = useL10n()
    const isDark = useComputedColorScheme() === 'dark'

    /** Compute navigation state based on the current active request and session. */
    const nav = useMemo<{
      readonly total: number
      readonly toFirst: string | null
      readonly toNewer: string | null
      readonly toOlder: string | null
      readonly toLast: string | null
    } | null>(() => {
      if (!sID) {
        return null
      }

      const rIDs = [...requests.values()]
      const idx = rID ? rIDs.findIndex((r) => r.id === rID) : -1

      const pathFor = (r: (typeof rIDs)[number] | undefined): string | null =>
        r ? pathTo(ROUTE_ID.Request, { sID: sID, rID: r.id }) : null

      return {
        total: rIDs.length,
        toFirst: idx > 0 ? pathFor(rIDs[0]) : null,
        toNewer: idx > 0 ? pathFor(rIDs[idx - 1]) : null,
        toOlder: idx !== -1 && idx < rIDs.length - 1 ? pathFor(rIDs[idx + 1]) : null,
        toLast: idx !== -1 && idx < rIDs.length - 1 ? pathFor(rIDs[rIDs.length - 1]) : null,
      }
    }, [requests, rID, sID])

    /** Listen for arrow keys to navigate between requests. */
    useEffect(() => {
      const handler = (e: KeyboardEvent): void => {
        if ((e.code === 'ArrowDown' || e.code === 'ArrowRight') && nav?.toOlder) {
          navigate(nav.toOlder)
        } else if ((e.code === 'ArrowUp' || e.code === 'ArrowLeft') && nav?.toNewer) {
          navigate(nav.toNewer)
        }
      }

      window.addEventListener('keydown', handler)

      return () => window.removeEventListener('keydown', handler)
    }, [nav, navigate])

    const shortProps: Partial<ButtonProps> = { variant: 'default', size: 'compact-xs', fw: 'inherit' }
    const longProps: Partial<ButtonProps> = { ...shortProps, styles: { section: { margin: 0 } } }

    return (
      <Group justify="space-between" wrap="nowrap" gap="xs" {...props}>
        <Button.Group miw={0}>
          <Button
            {...longProps}
            leftSection={<IconChevronsUp size="1em" />}
            title={t(L10nKey.firstRequest)}
            disabled={!nav?.toFirst}
            renderRoot={nav?.toFirst ? (rProps) => <Link to={nav?.toFirst} {...rProps} /> : undefined}
          />
          <Button
            {...shortProps}
            leftSection={<IconChevronUp size="1em" />}
            disabled={!nav?.toNewer}
            renderRoot={nav?.toNewer ? (rProps) => <Link to={nav?.toNewer} {...rProps} /> : undefined}
          >
            {t(L10nKey.newerRequest)}
          </Button>
        </Button.Group>
        <Badge color={isDark ? 'gray' : 'lime'} size="sm" px="0.7em" radius="sm">
          {nav?.total ?? 0}
        </Badge>
        <Button.Group miw={0}>
          <Button
            {...shortProps}
            rightSection={<IconChevronDown size="1em" />}
            disabled={!nav?.toOlder}
            renderRoot={nav?.toOlder ? (rProps) => <Link to={nav.toOlder} {...rProps} /> : undefined}
          >
            {t(L10nKey.olderRequest)}
          </Button>
          <Button
            {...longProps}
            leftSection={<IconChevronsDown size="1em" />}
            title={t(L10nKey.lastRequest)}
            disabled={!nav?.toLast}
            renderRoot={nav?.toLast ? (rProps) => <Link to={nav.toLast} {...rProps} /> : undefined}
          />
        </Button.Group>
      </Group>
    )
  },
  {
    Skeleton: ({ ...props }: SkeletonProps): React.JSX.Element => {
      return <Skeleton h={22} animate={false} {...props} />
    },
  }
)

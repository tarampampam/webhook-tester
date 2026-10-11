import type { CardProps, ElementProps, MantineSpacing, SkeletonProps } from '@mantine/core'
import { Badge, Card, CloseButton, Group, Skeleton, Text, useComputedColorScheme } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { clsx } from 'clsx'
import React, { memo, type PropsWithChildren, useCallback } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID } from '~/routing'
import { anyToError, isLocalhost, L10nKey, useL10n, useRequests, useUpdateEvery, When } from '~/shared'
import type { Request as RequestData } from '~/shared'
import { methodToColor, ThemeColor } from '~/theme'
import styles from './request.module.css'

export const Request = Object.assign(
  memo(function Request({
    request,
    active = true,
    children,
    ...props
  }: PropsWithChildren<{ request: RequestData; active?: boolean } & CardProps>): React.JSX.Element {
    const { t } = useL10n()
    const { requests, delRequest } = useRequests()
    const navigate = useNavigate()
    const isDark = useComputedColorScheme() === 'dark'

    const handleDelete = useCallback(
      (r: RequestData) => {
        const keys = [...requests.keys()]
        const idx = keys.indexOf(r.id)
        const [nextID, prevID] = [keys[idx + 1], keys[idx - 1]]

        delRequest(r.sID, r.id).catch((err) => {
          notifications.show({
            title: t(L10nKey.somethingWentWrong),
            message: anyToError(err).message,
            color: 'red',
          })
        })

        if (active) {
          if (nextID) {
            navigate(pathTo(ROUTE_ID.Request, { sID: r.sID, rID: nextID }))
          } else if (prevID) {
            navigate(pathTo(ROUTE_ID.Request, { sID: r.sID, rID: prevID }))
          } else {
            navigate(pathTo(ROUTE_ID.Session, { sID: r.sID }))
          }
        }
      },
      [active, delRequest, navigate, requests, t]
    )

    const cardBgColor = active
      ? isDark
        ? 'linear-gradient(90deg, var(--mantine-color-teal-filled), var(--mantine-color-cyan-filled))' // dark + active
        : undefined // light + active
      : isDark
        ? undefined // dark + inactive
        : `var(--mantine-color-${ThemeColor.NavbarBg}-8)` // light + inactive

    const cardColor = active
      ? isDark
        ? 'pure-white' // dark + active
        : undefined // light + active
      : isDark
        ? undefined // dark + inactive
        : 'white' // light + inactive

    const closeColor = active
      ? isDark
        ? 'white' // dark + active
        : undefined // light + active
      : isDark
        ? undefined // dark + inactive
        : 'white' // light + inactive

    // https://mantine.dev/guides/polymorphic/
    const asLinkProps: { component: typeof Link } & Pick<ElementProps<typeof Link>, 'to'> = {
      component: Link,
      to: pathTo(ROUTE_ID.Request, { sID: request.sID, rID: request.id }),
    }

    const isLocal = isLocalhost(request.clientAddress)

    // trick with paddings/margins is needed to increase the clickable area of the links without affecting the layout
    const innerPadding: MantineSpacing = 'xs'

    return (
      <Card
        style={{ background: cardBgColor }}
        p={0}
        pl={0}
        c={cardColor}
        className={clsx(styles.card, !active && (isDark ? styles.cardDark : styles.cardLight))}
        {...props}
      >
        <Group justify="space-between" gap={0}>
          <CloseButton
            variant="transparent"
            pos="absolute"
            right={2}
            top={2}
            size={16}
            c={closeColor}
            aria-label={t(L10nKey.delete)}
            title={t(L10nKey.delete)}
            onClick={() => handleDelete(request)}
          />
          <Text {...asLinkProps} size="xl" fw={500} flex={1} miw={0} pl={innerPadding} pt={innerPadding} truncate="end">
            <When condition={isLocal} wrapper={(children) => <span title={request.clientAddress}>{children}</span>}>
              {isLocal ? t(L10nKey.localhost) : request.clientAddress}
            </When>
          </Text>
          <Badge
            variant="dot"
            mx={0}
            radius="sm"
            mt={innerPadding}
            mr={innerPadding}
            size="sm"
            styles={{ label: { fontWeight: 300 } }}
            color={methodToColor(request.method)}
          >
            {request.method}
          </Badge>
        </Group>
        <Group gap="0.5ch" wrap="nowrap" fz="xs">
          <Text {...asLinkProps} pl={innerPadding} pb={innerPadding} fz="1em" truncate="end">
            <ExactTime capturedAt={request.capturedAt} />
          </Text>
          <Text {...asLinkProps} pr={innerPadding} pb={innerPadding} flex={1} fz="0.8em" truncate="end">
            <RelativeTime capturedAt={request.capturedAt} />
          </Text>
        </Group>
        {children}
      </Card>
    )
  }),
  {
    Skeleton: ({ count = 1, ...props }: { count?: number } & SkeletonProps): React.JSX.Element => {
      return (
        <>
          {Array.from({ length: count }).map((_, i) => (
            <Skeleton key={i} h={71.6} animate={false} {...props} />
          ))}
        </>
      )
    },
  }
)

/**
 * Displays the exact time of the request, updating every second to keep the fractional seconds accurate.
 *
 * Declared as a separate component to optimize performance, to re-render only it instead of the entire parent
 * component N ms.
 */
const ExactTime = ({ capturedAt, interval = 1000 }: { capturedAt: Date; interval?: number }): React.JSX.Element => {
  const get = useCallback(() => {
    const h = capturedAt.getHours().toString().padStart(2, '0')
    const m = capturedAt.getMinutes().toString().padStart(2, '0')
    const s = capturedAt.getSeconds().toString().padStart(2, '0')
    const ms = Math.floor(capturedAt.getMilliseconds() / 10)
      .toString()
      .padStart(2, '0')

    return (
      <>
        {h}:{m}:{s}
        <span style={{ opacity: 0.7, fontSize: '0.9em' }}>.{ms}</span>
      </>
    )
  }, [capturedAt])
  const value = useUpdateEvery(get, interval)

  return <>{value}</>
}

/**
 * Displays the relative time from the request, updating every 100 ms to keep it accurate.
 *
 * Declared as a separate component to optimize performance, to re-render only it instead of the entire parent
 * component N ms.
 */
const RelativeTime = ({ capturedAt, interval = 100 }: { capturedAt: Date; interval?: number }): React.JSX.Element => {
  const { relativeTime } = useL10n()
  const get = useCallback(() => relativeTime(capturedAt, new Date()).toLowerCase(), [relativeTime, capturedAt])
  const value = useUpdateEvery(get, interval)

  return <span style={{ opacity: 0.7 }}>({value})</span>
}

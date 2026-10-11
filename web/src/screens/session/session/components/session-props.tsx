import type { DataListItemValueProps, DataListProps } from '@mantine/core'
import { Badge, DataList, Group } from '@mantine/core'
import React, { useEffect, useState } from 'react'
import { humanizeBytes, L10nKey, type Session, useL10n } from '~/shared'
import { statusCodeToColor } from '~/theme'

export const SessionProps = ({
  session,
  alignRight = true,
  ...props
}: {
  session: Session
  alignRight?: boolean
} & DataListProps): React.JSX.Element => {
  const { t } = useL10n()
  const [bodyType, setBodyType] = useState<'text' | 'binary'>('binary')
  const [bodyLength, setBodyLength] = useState<number>(0)

  useEffect(() => {
    session.response.getBody().then((b) => {
      if (b === null) {
        return
      }

      try {
        // try to decode the body as UTF-8 text, and treat it as binary if it contains null bytes or if it
        // fails to decode
        setBodyType(new TextDecoder('utf-8', { fatal: true }).decode(b).includes('\0') ? 'binary' : 'text')
      } catch {
        setBodyType('binary')
      } finally {
        setBodyLength(b.length)
      }
    })
  }, [session])

  const valueProps: DataListItemValueProps = { w: '100%', my: 5 }

  const headers = {
    list: session.response.headers,
    limit: 5,
    hidden: session.response.headers.length > 5 ? session.response.headers.slice(5) : [],
  }

  return (
    <DataList orientation="vertical" ta={alignRight ? 'right' : undefined} {...props}>
      <DataList.Item>
        <DataList.ItemLabel>{t(L10nKey.statusCode)}</DataList.ItemLabel>
        <DataList.ItemValue {...valueProps}>
          <Badge color={statusCodeToColor(session.response.code)}>HTTP {session.response.code.toString()}</Badge>
        </DataList.ItemValue>
      </DataList.Item>
      <DataList.Item>
        <DataList.ItemLabel>{t(L10nKey.responseDelay)}</DataList.ItemLabel>
        <DataList.ItemValue {...valueProps}>
          <Badge color="gray" tt="none">
            {session.response.delay > 0
              ? `${session.response.delay.toFixed(3)} ${t(L10nKey.timeSec)}`
              : t(L10nKey.noDelay)}
          </Badge>
        </DataList.ItemValue>
      </DataList.Item>
      <DataList.Item>
        <DataList.ItemLabel>{t(L10nKey.responseHeaders)}</DataList.ItemLabel>
        <DataList.ItemValue ta="inherit" {...valueProps}>
          <Group justify={alignRight ? 'flex-end' : 'flex-start'} gap={6}>
            {headers.list.length === 0 && (
              <Badge color="gray" tt="none">
                {t(L10nKey.noHeaders)}
              </Badge>
            )}
            {headers.list.slice(0, headers.limit).map(({ name, value }) => (
              <Badge key={name} color="blue" title={value} tt="none">
                {name}
              </Badge>
            ))}
            {headers.list.length > headers.limit && (
              <Badge color="blue" title={headers.hidden.map(({ name }) => name).join(', ')} tt="none">
                +{headers.hidden.length} {t(L10nKey.plusMore)}
              </Badge>
            )}
          </Group>
        </DataList.ItemValue>
      </DataList.Item>
      <DataList.Item>
        <DataList.ItemLabel>{t(L10nKey.responseBody)}</DataList.ItemLabel>
        <DataList.ItemValue {...valueProps}>
          <Badge color="gray" tt="none">
            {humanizeBytes(bodyLength)} ({bodyType})
          </Badge>
        </DataList.ItemValue>
      </DataList.Item>
    </DataList>
  )
}

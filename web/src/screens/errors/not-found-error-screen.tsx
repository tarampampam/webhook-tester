import { Button, Center, Stack, Text, Title } from '@mantine/core'
import type React from 'react'
import { Link } from 'react-router-dom'
import { pathTo, ROUTE_ID } from '~/routing'
import { L10nKey, useL10n } from '~/shared'

export const NotFoundErrorScreen = (): React.JSX.Element => {
  const { t } = useL10n()

  return (
    <Center maw="100%" h="100%">
      <Stack align="center">
        <Title size="10em" style={{ fontFamily: 'monospace' }}>
          404
        </Title>
        <Stack align="center">
          <Text size="lg">{t(L10nKey.notFoundMessage)}</Text>
          <Link to={pathTo(ROUTE_ID.Home)}>
            <Button variant="filled" size="xs">
              {t(L10nKey.goToHome)}
            </Button>
          </Link>
        </Stack>
      </Stack>
    </Center>
  )
}

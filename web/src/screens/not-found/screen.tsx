import type React from 'react'
import { Center, Text, Button, Title, Stack } from '@mantine/core'
import { Link } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'

export default function Screen(): React.JSX.Element {
  return (
    <Center maw="100%" h="100%">
      <Stack align="center">
        <Title size="10em" style={{ fontFamily: 'monospace' }}>
          404
        </Title>
        <Stack align="center">
          <Text size="lg">The requested page was not found</Text>
          <Link to={pathTo(RouteIDs.Home)}>
            <Button variant="filled" size="xs">
              Go to home
            </Button>
          </Link>
        </Stack>
      </Stack>
    </Center>
  )
}

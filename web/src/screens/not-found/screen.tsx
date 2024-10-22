import type React from 'react'
import { Center, Text, Button } from '@mantine/core'
import { Link } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'

export default function Screen(): React.JSX.Element {
  return (
    <Center maw="100%" h="100%">
      <div>
        <Text size="lg">The requested page was not found</Text>
        <Center pt="1em">
          <Link to={pathTo(RouteIDs.Home)}>
            <Button variant="filled" size="xs">
              Go to home
            </Button>
          </Link>
        </Center>
      </div>
    </Center>
  )
}

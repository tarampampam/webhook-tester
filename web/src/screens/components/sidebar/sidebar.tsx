import type React from 'react'
import { Button, Center, Image, Loader, Stack, Text } from '@mantine/core'
import { IconTrash } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { pathTo, RouteIDs } from '~/routing'
import { Request, Navigator } from './components'
import PandaSvg from '~/assets/panda.svg'
import { useData } from '~/shared'

export const SideBar = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { session, request, requests, removeAllRequests } = useData()

  return (
    <Stack align="stretch" justify="flex-start" gap="xs">
      {(!!session &&
        ((!!requests.length && (
          <>
            <Navigator />

            {requests.map((rq) => (
              <Request sID={session.sID} request={rq} key={rq.rID} isActive={!!request && request.rID === rq.rID} />
            ))}

            {requests.length > 1 && (
              <Center>
                <Button
                  leftSection={<IconTrash size="1em" />}
                  size="compact-xs"
                  variant="outline"
                  color="red"
                  px="xs"
                  mb="sm"
                  radius="xl"
                  opacity={0.7}
                  onClick={() => {
                    removeAllRequests(session.sID)
                      .then((slow) => slow())
                      .then(() =>
                        // navigate to the session screen
                        navigate(pathTo(RouteIDs.SessionAndRequest, session.sID))
                      )
                  }}
                >
                  Delete all requests
                </Button>
              </Center>
            )}
          </>
        )) || <NoRequests />)) || <NoSession />}
    </Stack>
  )
}

const NoRequests = (): React.JSX.Element => (
  <Stack gap="xs" h="100%" justify="space-between">
    <Center pt="2em">
      <Image src={PandaSvg} w="50%" />
    </Center>
    <Center>
      <Loader color="dimmed" size="1em" mr={8} mb={3} />
      <Text c="dimmed">Waiting for first request</Text>
    </Center>
  </Stack>
)

const NoSession = (): React.JSX.Element => (
  <Center pt="2em">
    <Loader color="dimmed" size="1em" mr={8} mb={3} />
    <Text c="dimmed">No session selected</Text>
  </Center>
)

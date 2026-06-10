import React, { useRef } from 'react'
import { Button, Center, Loader, Stack, Text } from '@mantine/core'
import { IconTrash } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveSessionID } from '~/routing'
import { Request, Navigator } from './components'
import { ImagePanda, useRequests, useSessions } from '~/shared'

export const SideBar = (): React.JSX.Element => {
  const navigate = useNavigate()
  const { sessions } = useSessions()
  const { requests, delAllRequests } = useRequests()
  const sID = useActiveSessionID()
  const activeRequestRef = useRef<HTMLDivElement>(null)

  return (
    <Stack align="stretch" justify="flex-start" gap="xs">
      {(!!sessions.size &&
        ((!!requests.size && sID && (
          <>
            <Navigator />

            {Array.from(requests, ([rID, rq]) => (
              <Request
                sID={sID}
                request={rq}
                key={rID}
                isActive={rID === rq.id}
                componentRef={rID === rq.id ? activeRequestRef : null}
              />
            ))}

            {requests.size > 1 && (
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
                    delAllRequests(sID).then(() =>
                      // navigate to the session screen
                      navigate(pathTo(ROUTE_ID.SessionAndRequest, { sID: sID }))
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
      <ImagePanda w="50%" />
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

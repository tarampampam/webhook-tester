import type React from 'react'
import { useNavigate, useRouteError } from 'react-router-dom'
import { Alert, Center } from '@mantine/core'
import { IconBug } from '@tabler/icons-react'

export const RuntimeErrorScreen = (): React.JSX.Element => {
  const navigate = useNavigate()
  const error = useRouteError()

  let message = error instanceof Error ? error.message : String(error)
  message = message.charAt(0).toUpperCase() + message.slice(1)

  return (
    <Center maw="100%" h="100%">
      <Alert
        variant="filled"
        color="red"
        radius="md"
        my="lg"
        title={message}
        icon={<IconBug />}
        onClose={() => navigate(-1)}
        withCloseButton
      >
        Please, report this issue to the developers. You can find more details in the console log.
      </Alert>
    </Center>
  )
}

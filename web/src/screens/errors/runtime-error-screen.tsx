import { Alert, Center } from '@mantine/core'
import { IconBug } from '@tabler/icons-react'
import React from 'react'
import { useLocation, useNavigate, useRouteError } from 'react-router-dom'
import { L10nKey, useL10n } from '~/shared'

export const RuntimeErrorScreen = (): React.JSX.Element => {
  const navigate = useNavigate()
  const location = useLocation()
  const error = useRouteError()
  const { t } = useL10n()
  const canGoBack = location.key !== 'default'

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
        onClose={canGoBack ? () => navigate(-1) : undefined}
        withCloseButton={canGoBack}
      >
        {t(L10nKey.reportIssueToDevelopers)}. {t(L10nKey.moreDetailsInConsole)}.
      </Alert>
    </Center>
  )
}

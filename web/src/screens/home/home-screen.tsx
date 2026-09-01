import React, { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { Alert, Center, Loader, Stack, Text } from '@mantine/core'
import { ROUTE_ID, pathTo } from '~/routing'
import { anyToError, L10nKey, useL10n, useLastUsed, useSessions } from '~/shared'

// HomeGuard already redirected when lastUsedSID was present, so this screen only renders when
// there are no known sessions - the only thing left to do is create a fresh one and go there
export const HomeScreen = (): React.JSX.Element => {
  const { setLastUsedSID } = useLastUsed()
  const { newSession } = useSessions()
  const { t } = useL10n()
  const [state, setState] = useState<
    { phase: 'loading' } | { phase: 'redirect'; to: string } | { phase: 'error'; message: string }
  >({ phase: 'loading' })

  useEffect(() => {
    const ctrl = new AbortController()

    void (async () => {
      try {
        const sID = await newSession({}, { signal: ctrl.signal })
        if (ctrl.signal.aborted) {
          return
        }

        // persist the new session as last-used so HomeGuard sends the user here on next visit
        setLastUsedSID(sID)
        setState({ phase: 'redirect', to: pathTo(ROUTE_ID.Session, { sID }) })
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        setState({ phase: 'error', message: anyToError(err).message })
      }
    })()

    // cancel the in-flight API call on cleanup (component unmounted or effect re-ran)
    return () => ctrl.abort()
  }, [newSession, setLastUsedSID])

  switch (state.phase) {
    case 'loading':
      return (
        <Stack h="100%" align="center" justify="center" gap={0}>
          <Loader color="cyan" mb="md" />
          <Text fz="0.7em" c="dimmed">
            {t(L10nKey.pleaseWait)}
          </Text>
          <Text fz="0.7em" c="dimmed">
            {t(L10nKey.creatingSession)}…
          </Text>
        </Stack>
      )

    case 'redirect':
      return <Navigate to={state.to} replace />

    case 'error':
      return (
        <Center h="100%">
          <Alert color="red" title={t(L10nKey.somethingWentWrong)}>
            {state.message}
          </Alert>
        </Center>
      )

    // will never happen, but added for consistency
    default:
      throw new Error('Unhandled HomeScreen state phase')
  }
}

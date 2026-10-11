import { Alert, Center, Loader, Stack, Text } from '@mantine/core'
import React, { useEffect, useRef, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveSessionID } from '~/routing'
import { anyToError, L10nKey, useL10n, useLastUsed, useSessions } from '~/shared'

export const SessionGuard = ({ children }: { children: React.ReactNode }): React.JSX.Element => {
  const { sessions, isReady, addExistingSession, newSession } = useSessions()
  const { lastUsedSID, setLastUsedSID } = useLastUsed()
  const { t } = useL10n()
  const [state, setState] = useState<
    | { phase: 'loading'; step: 'sessions' | 'switching' | 'creating' }
    | { phase: 'redirect'; to: string }
    | { phase: 'error'; message: string }
  >({ phase: 'loading', step: 'sessions' })
  const processedSIDRef = useRef<string | null>(null)
  const sID = useActiveSessionID()
  if (!sID) {
    throw new Error('SessionGuard must be used within a session route')
  }

  // persist last-used session whenever we land on a valid one - covers both the happy path (already in sessions map)
  // and the case where addExistingSession just confirmed it
  useEffect(() => {
    if (isReady && sID && sessions.has(sID)) {
      setLastUsedSID(sID)
    }
  }, [isReady, sID, sessions, setLastUsedSID])

  useEffect(() => {
    // wait for SessionsProvider to finish hydrating from IndexedDB and validating against the backend
    if (!isReady) {
      return
    }

    // happy path - session is already known and backend-validated - render derives this directly from
    // sessions.has(sID), no state transition needed
    if (sessions.has(sID)) {
      return
    }

    // already committed to handling this sID - prevent re-firing when sessions map updates after the decision
    if (processedSIDRef.current === sID) {
      return
    }
    processedSIDRef.current = sID

    const ctrl = new AbortController()

    setState({ phase: 'loading', step: 'switching' })

    // session is unknown (e.g. user opened a shared link) - ask the backend whether it exists
    void (async () => {
      try {
        const exists = await addExistingSession(sID, { signal: ctrl.signal })
        if (ctrl.signal.aborted) {
          return
        }

        if (exists) {
          // backend confirmed the session - sessions map now has sID, render will show children automatically,
          // setLastUsedSID effect will persist it
          return
        }

        // session doesn't exist on the backend - find the best redirect target. all candidates below come from the
        // already-validated sessions map, so we know they're live and won't trigger another round of "not found"

        // prefer the last session the user actually used
        if (lastUsedSID && lastUsedSID !== sID && sessions.has(lastUsedSID)) {
          setState({ phase: 'redirect', to: pathTo(ROUTE_ID.Session, { sID: lastUsedSID }) })

          return
        }

        // fall back to whatever session we know about
        const firstKnown = [...sessions.keys()].find((id) => id !== sID)
        if (firstKnown) {
          setState({ phase: 'redirect', to: pathTo(ROUTE_ID.Session, { sID: firstKnown }) })

          return
        }

        setState({ phase: 'loading', step: 'creating' })

        // no sessions at all - create a fresh one and land there
        const newSID = await newSession({}, { signal: ctrl.signal })
        if (ctrl.signal.aborted) {
          return
        }

        setLastUsedSID(newSID)
        setState({ phase: 'redirect', to: pathTo(ROUTE_ID.Session, { sID: newSID }) })
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }

        setState({ phase: 'error', message: anyToError(err).message })
      }
    })()

    // cancel any in-flight API call on cleanup (meaningful when sID changes or component unmounts)
    return () => ctrl.abort()
  }, [sID, isReady, sessions, lastUsedSID, addExistingSession, newSession, setLastUsedSID])

  // happy path - derived directly from the sessions map - no async check or state needed
  if (isReady && sID && sessions.has(sID)) {
    return <>{children}</>
  }

  switch (state.phase) {
    // waiting - sessions provider not ready yet, or async calls in progress
    case 'loading': {
      const msg = ((): string => {
        switch (state.step) {
          case 'sessions':
            return t(L10nKey.loadingSessions)

          case 'switching':
            return t(L10nKey.switchingSession)

          case 'creating':
            return t(L10nKey.creatingSession)
        }
      })()

      return (
        <Stack h="100%" align="center" justify="center" gap={0}>
          <Loader color="cyan" mb="md" />
          <Text fz="0.7em" c="dimmed">
            {t(L10nKey.pleaseWait)}
          </Text>
          <Text fz="0.7em" c="dimmed">
            {msg}…
          </Text>
        </Stack>
      )
    }

    // redirect to the best known session
    case 'redirect':
      return <Navigate to={state.to} replace />

    // show an error if the async check failed (e.g. network error, backend down, etc.)
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
      throw new Error('Unhandled SessionGuard state phase')
  }
}

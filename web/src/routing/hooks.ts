import { useMatches } from 'react-router-dom'

/**
 * Returns the session ID from the currently matched route, or null when no session route is active.
 *
 * Re-renders only on navigation - safe to use as a useEffect dependency.
 */
export const useActiveSessionID = (): string | null => {
  const matches = useMatches()
  return matches.find((m) => m.id === 'session')?.params?.sID ?? null
}

/**
 * Returns the request ID from the currently matched route, or null when no request is selected.
 *
 * Re-renders only on navigation - safe to use as a useEffect dependency.
 */
export const useActiveRequestID = (): string | null => {
  const matches = useMatches()
  return matches.find((m) => m.id === 'request')?.params?.rID ?? null
}

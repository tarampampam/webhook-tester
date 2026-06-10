import { Navigate, useMatches, type RouteObject } from 'react-router-dom'
import { DefaultLayout } from '~/screens'
import { NotFoundScreen } from '~/screens/errors/not-found'
import { RuntimeErrorScreen } from '~/screens/errors/runtime-error'
import { HomeScreen } from '~/screens/home'
import { SessionAndRequestScreen } from '~/screens/session'

/** The route IDs used in the app. */
export enum ROUTE_ID {
  Home = 'home',
  SessionAndRequest = 'session-and-request',
}

const ROUTE_PATTERNS: Record<ROUTE_ID, string> = {
  [ROUTE_ID.Home]: '/',
  [ROUTE_ID.SessionAndRequest]: 's/:sID/:rID?',
}

type RouteParamsMap = {
  [ROUTE_ID.Home]: never
  [ROUTE_ID.SessionAndRequest]: { sID: string; rID?: string }
}

export const createRoutes = (): RouteObject[] => [
  {
    path: '/',
    element: <DefaultLayout />,
    errorElement: <RuntimeErrorScreen />,
    children: [
      {
        index: true,
        id: ROUTE_ID.Home,
        element: <HomeScreen />,
      },
      {
        path: 's/',
        element: <Navigate to={pathTo(ROUTE_ID.Home)} replace />,
      },
      {
        path: ROUTE_PATTERNS[ROUTE_ID.SessionAndRequest],
        id: ROUTE_ID.SessionAndRequest,
        element: <SessionAndRequestScreen />,
      },
      {
        path: '*',
        element: <NotFoundScreen />,
      },
    ],
  },
]

/**
 * Converts a route ID to a path to use in a link.
 *
 * @example
 * ```tsx
 * <Link to={pathTo(ROUTE_ID.Home)}>Go to home</Link>
 * <Link to={pathTo(ROUTE_ID.SessionAndRequest, { sID: 'abc' })}>Open session</Link>
 * ```
 */
export const pathTo = <T extends ROUTE_ID>(
  id: T,
  ...args: RouteParamsMap[T] extends never ? [] : [RouteParamsMap[T]]
): string => {
  switch (id) {
    case ROUTE_ID.Home:
      return '/'
    case ROUTE_ID.SessionAndRequest: {
      // TS cannot narrow conditional rest args inside a switch on a generic param
      const { sID, rID } = args[0] as RouteParamsMap[ROUTE_ID.SessionAndRequest]

      if (!rID) {
        return `/s/${encodeURIComponent(sID)}`
      }

      return `/s/${encodeURIComponent(sID)}/${encodeURIComponent(rID)}`
    }
    default:
      throw new Error(`Unknown route: ${String(id)}`)
  }
}

/**
 * Returns the session ID from the currently matched route, or null when no session route is active.
 *
 * Re-renders only on navigation - safe to use as a useEffect dependency.
 */
export const useActiveSessionID = (): string | null => {
  const matches = useMatches()
  return matches.find((m) => m.id === ROUTE_ID.SessionAndRequest)?.params?.sID ?? null
}

/**
 * Returns the request ID from the currently matched route, or null when no request is selected.
 *
 * Re-renders only on navigation - safe to use as a useEffect dependency.
 */
export const useActiveRequestID = (): string | null => {
  const matches = useMatches()
  return matches.find((m) => m.id === ROUTE_ID.SessionAndRequest)?.params?.rID ?? null
}

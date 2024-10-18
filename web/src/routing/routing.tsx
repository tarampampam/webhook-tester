import { createPath, matchRoutes, type RouteObject, useLocation } from 'react-router-dom'
import { HomeLayout } from '~/screens'
import { NotFoundScreen } from '~/screens/not-found'
import { NoSessionScreen, SessionLayout } from '~/screens/session'
import { SessionRequestScreen } from '~/screens/session/request'
import { apiClient } from '~/api'

export enum RouteIDs {
  Home = 'home',
  NoSession = 'no-session',
  Session = 'session',
  SessionNotFound = 'session.404', // TODO: implement
  SessionRequest = 'request',
  SessionRequestNotFound = 'request.404', // TODO: implement
  NotFound = '404', // TODO: use it
}

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <HomeLayout apiClient={apiClient} />,
    errorElement: <NotFoundScreen />,
    id: RouteIDs.Home,
    children: [
      {
        index: true,
        element: <NoSessionScreen />,
        id: RouteIDs.NoSession,
      },
    ],
  },
  {
    path: 'session/:sID',
    id: RouteIDs.Session,
    element: <SessionLayout />,
    children: [
      {
        path: ':rID',
        id: RouteIDs.SessionRequest,
        element: <SessionRequestScreen />,
      },
    ],
  },
]

/** Resolves the current route ID from the router. */
export function useCurrentRouteID(): RouteIDs | undefined {
  const match = matchRoutes(routes, useLocation())

  if (match) {
    const ids = Object.values<string>(RouteIDs)

    for (const route of match.reverse()) {
      if (route.route.id && ids.includes(route.route.id)) {
        return route.route.id as RouteIDs
      }
    }
  }

  return undefined
}

type RouteParams<T extends RouteIDs> = T extends RouteIDs.Session
  ? [string] // sID
  : T extends RouteIDs.SessionRequest
    ? [string, string] // sID, rID
    : [] // no params

/**
 * Converts a route ID to a path to use in a link.
 *
 * @example
 * ```tsx
 * <Link to={pathTo(RouteIDs.Home)}>Go to home</Link>
 * ```
 */
export function pathTo<T extends RouteIDs>(
  path: RouteIDs,
  ...params: T extends RouteIDs
    ? RouteParams<Exclude<T, RouteIDs.SessionNotFound | RouteIDs.SessionRequestNotFound | RouteIDs.NotFound>>
    : never
): string {
  switch (path) {
    case RouteIDs.Home:
      return createPath({ pathname: '/' })
    case RouteIDs.Session: {
      const sID = encodeURIComponent(params[0] ?? '')

      return createPath({ pathname: `/session/${sID}` })
    }
    case RouteIDs.SessionRequest: {
      const sID = encodeURIComponent(params[0] ?? '')
      const rID = encodeURIComponent(params[1] ?? '')

      return createPath({ pathname: `/session/${sID}/${rID}` })
    }
    default:
      throw new Error(`Unknown route: ${path}`) // will never happen because of the type guard
  }
}

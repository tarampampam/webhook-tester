import { createPath, type RouteObject } from 'react-router-dom'
import { apiClient } from '~/api'
import { DefaultLayout } from '~/screens'
import { NotFoundScreen } from '~/screens/not-found'
import { SessionAndRequestScreen } from '~/screens/session'
import { HomeScreen } from '~/screens/home'

export enum RouteIDs {
  Home = 'home',
  SessionAndRequest = 'session-and-request',
}

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <DefaultLayout apiClient={apiClient} />,
    errorElement: <NotFoundScreen />,
    children: [
      {
        index: true,
        element: <HomeScreen apiClient={apiClient} />,
        id: RouteIDs.Home,
      },
      {
        // please note that `sID` and `rID` accessed via `useParams` hook, and changing this will break the app
        path: 's/:sID/:rID?',
        id: RouteIDs.SessionAndRequest,
        element: <SessionAndRequestScreen apiClient={apiClient} />,
      },
    ],
  },
]

type RouteParams<T extends RouteIDs> = T extends RouteIDs.SessionAndRequest
  ? [string /* sID */, string? /* rID (optional) */]
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
  ...params: T extends RouteIDs ? RouteParams<T> : never
): string {
  switch (path) {
    case RouteIDs.Home:
      return createPath({ pathname: '/' })
    case RouteIDs.SessionAndRequest: {
      const [sID, rID] = [params[0] ?? 'no-session', params[1]]

      if (!rID) {
        return createPath({ pathname: `/s/${encodeURIComponent(sID)}` })
      }

      return createPath({ pathname: `/s/${encodeURIComponent(sID)}/${encodeURIComponent(rID)}` })
    }
    default:
      throw new Error(`Unknown route: ${path}`) // will never happen because of the type guard
  }
}

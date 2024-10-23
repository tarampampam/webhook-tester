import type React from 'react'
import { createPath, matchRoutes, Navigate, type RouteObject, useLocation } from 'react-router-dom'
import { apiClient } from '~/api'
import { HomeLayout } from '~/screens'
import { NotFoundScreen } from '~/screens/not-found'
import { SessionLayout } from '~/screens/session'
import { SessionRequestScreen } from '~/screens/session/request'
import { HomeScreen } from '~/screens/home'
import { useLaseUsedRID, useLaseUsedSID } from '../shared'

export enum RouteIDs {
  Home = 'home',
  Session = 'session',
  SessionRequest = 'request',
}

const RedirectIfLastUsedIsKnown = ({ children }: { children: React.JSX.Element }): React.JSX.Element => {
  const [sID, rID] = [useLaseUsedSID()[0], useLaseUsedRID()[0]]

  if (sID && rID) {
    return <Navigate to={pathTo(RouteIDs.SessionRequest, sID, rID)} />
  } else if (sID) {
    return <Navigate to={pathTo(RouteIDs.Session, sID)} />
  }

  return children
}

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <HomeLayout apiClient={apiClient} />,
    errorElement: <NotFoundScreen />,
    children: [
      {
        index: true,
        element: (
          <RedirectIfLastUsedIsKnown>
            <HomeScreen />
          </RedirectIfLastUsedIsKnown>
        ),
        id: RouteIDs.Home,
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
  ? [string /* sID */]
  : T extends RouteIDs.SessionRequest
    ? [string /* sID */, string /* rID */]
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

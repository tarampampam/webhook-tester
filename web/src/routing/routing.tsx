import { Navigate, type RouteObject } from 'react-router-dom'
import { DefaultLayout } from '~/screens/layout'
import { NotFoundErrorScreen } from '~/screens/errors'
import { RuntimeErrorScreen } from '~/screens/errors'
import { HomeScreen } from '~/screens/home'
import { SessionLayout } from '~/screens/session'
import { SessionScreen } from '~/screens/session/session'
import { RequestScreen } from '~/screens/session/request'
import { WebhookURLProvider } from '../shared'
import { HomeGuard, SessionGuard } from './guards'
export { useActiveSessionID, useActiveRequestID } from './hooks'

/** The route IDs used in the app. */
export enum ROUTE_ID {
  Home = 'home',
  Session = 'session',
  Request = 'request',
}

const ROUTE_PATTERNS: Record<ROUTE_ID, string> = {
  [ROUTE_ID.Home]: '/',
  [ROUTE_ID.Session]: 's/:sID',
  [ROUTE_ID.Request]: ':rID',
}

type RouteParamsMap = {
  [ROUTE_ID.Home]: never
  [ROUTE_ID.Session]: { sID: string }
  [ROUTE_ID.Request]: { sID: string; rID: string }
}

export const createRoutes = (): RouteObject[] => [
  {
    path: '/',
    element: (
      <WebhookURLProvider.WithConfig>
        <DefaultLayout />
      </WebhookURLProvider.WithConfig>
    ),
    errorElement: <RuntimeErrorScreen />,
    children: [
      {
        index: true,
        id: ROUTE_ID.Home,
        element: (
          <HomeGuard>
            <HomeScreen />
          </HomeGuard>
        ),
      },
      {
        path: 's/',
        element: <Navigate to={pathTo(ROUTE_ID.Home)} replace />,
      },
      {
        path: ROUTE_PATTERNS[ROUTE_ID.Session],
        id: ROUTE_ID.Session,
        element: (
          <SessionGuard>
            <SessionLayout />
          </SessionGuard>
        ),
        children: [
          {
            index: true,
            element: <SessionScreen />,
          },
          {
            path: ROUTE_PATTERNS[ROUTE_ID.Request],
            id: ROUTE_ID.Request,
            element: <RequestScreen />,
          },
        ],
      },
      {
        path: '*',
        element: <NotFoundErrorScreen />,
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
 * <Link to={pathTo(ROUTE_ID.Session, { sID: 'abc' })}>Open session</Link>
 * <Link to={pathTo(ROUTE_ID.Request, { sID: 'abc', rID: 'def' })}>Open request</Link>
 * ```
 */
export const pathTo = <T extends ROUTE_ID>(
  id: T,
  ...args: RouteParamsMap[T] extends never ? [] : [RouteParamsMap[T]]
): string => {
  switch (id) {
    case ROUTE_ID.Home:
      return '/'
    case ROUTE_ID.Session: {
      const { sID } = args[0] as RouteParamsMap[ROUTE_ID.Session]

      return `/s/${encodeURIComponent(sID)}`
    }
    case ROUTE_ID.Request: {
      const { sID, rID } = args[0] as RouteParamsMap[ROUTE_ID.Request]

      return `/s/${encodeURIComponent(sID)}/${encodeURIComponent(rID)}`
    }
    default:
      throw new Error(`Unknown route: ${String(id)}`)
  }
}


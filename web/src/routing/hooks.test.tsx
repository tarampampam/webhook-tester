import { render } from '@testing-library/react'
import { type FC } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { describe, expect, test } from 'vitest'
import { ROUTE_ID, useActiveRequestID, useActiveSessionID } from './routing'

/** Builds a data router for a given path, rendering Capture component on every route. */
const makeRouter = (path: string, Capture: FC) =>
  createMemoryRouter(
    [
      {
        path: '/s/:sID',
        id: ROUTE_ID.Session,
        element: <Capture />,
        children: [
          { index: true, element: null },
          { path: ':rID', id: ROUTE_ID.Request, element: <Capture /> },
        ],
      },
      { path: '/', element: <Capture /> },
      { path: '*', element: <Capture /> },
    ],
    { initialEntries: [path] }
  )

describe('useActiveSessionID', () => {
  test.each<{ path: string; expected: string | null }>([
    { path: '/s/abc', expected: 'abc' },
    { path: '/s/abc/def', expected: 'abc' },
    { path: '/', expected: null },
    { path: '/other', expected: null },
  ])('$path → $expected', ({ path, expected }) => {
    let result: string | null = null
    const Capture: FC = () => {
      result = useActiveSessionID()
      return null
    }
    render(<RouterProvider router={makeRouter(path, Capture)} />)
    expect(result).toBe(expected)
  })
})

describe('useActiveRequestID', () => {
  test.each<{ path: string; expected: string | null }>([
    { path: '/s/abc/def', expected: 'def' },
    { path: '/s/abc', expected: null },
    { path: '/', expected: null },
    { path: '/other', expected: null },
  ])('$path → $expected', ({ path, expected }) => {
    let result: string | null = null
    const Capture: FC = () => {
      result = useActiveRequestID()
      return null
    }
    render(<RouterProvider router={makeRouter(path, Capture)} />)
    expect(result).toBe(expected)
  })
})

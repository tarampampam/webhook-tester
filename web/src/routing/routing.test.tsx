import { render } from '@testing-library/react'
import { type FC, isValidElement } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { describe, expect, test } from 'vitest'
import { createRoutes, pathTo, ROUTE_ID, useActiveRequestID, useActiveSessionID } from './routing'

/** Builds a data router for a given path, rendering Capture component on every route. */
const makeRouter = (path: string, Capture: FC) =>
  createMemoryRouter(
    [
      { path: '/s/:sID/:rID?', id: ROUTE_ID.SessionAndRequest, element: <Capture /> },
      { path: '/', element: <Capture /> },
      { path: '*', element: <Capture /> },
    ],
    { initialEntries: [path] }
  )

describe('pathTo', () => {
  describe(ROUTE_ID.Home, () => {
    test('returns /', () => {
      expect(pathTo(ROUTE_ID.Home)).toBe('/')
    })
  })

  describe(ROUTE_ID.SessionAndRequest, () => {
    test.each<{ params: { sID: string; rID?: string }; expected: string }>([
      { params: { sID: 'abc' }, expected: '/s/abc' },
      { params: { sID: 'abc', rID: 'def' }, expected: '/s/abc/def' },
      { params: { sID: 'abc', rID: undefined }, expected: '/s/abc' },
      { params: { sID: 'foo/bar' }, expected: '/s/foo%2Fbar' },
      { params: { sID: 'abc', rID: 'x/y' }, expected: '/s/abc/x%2Fy' },
    ])('$expected', ({ params, expected }) => {
      expect(pathTo(ROUTE_ID.SessionAndRequest, params)).toBe(expected)
    })
  })
})

describe('createRoutes', () => {
  const children = createRoutes()[0]?.children ?? []

  test.each([
    { id: ROUTE_ID.Home, index: true },
    { id: ROUTE_ID.SessionAndRequest, path: 's/:sID/:rID?' },
  ])('route $id is registered', ({ id, ...shape }) => {
    expect(children.find((r) => r.id === id)).toMatchObject(shape)
  })

  test('wildcard 404 route is present', () => {
    expect(children.find((r) => r.path === '*')).toBeDefined()
  })

  test('s/ redirects to home with replace', () => {
    const element = children.find((r) => r.path === 's/')?.element
    expect(isValidElement(element)).toBe(true)
    if (isValidElement<{ to: string; replace: boolean }>(element)) {
      expect(element.props.to).toBe(pathTo(ROUTE_ID.Home))
      expect(element.props.replace).toBe(true)
    }
  })
})

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

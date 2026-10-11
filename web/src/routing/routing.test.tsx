import { isValidElement } from 'react'
import { describe, expect, test } from 'vitest'
import { createRoutes, pathTo, ROUTE_ID } from './routing'

describe('pathTo', () => {
  describe(ROUTE_ID.Home, () => {
    test('returns /', () => {
      expect(pathTo(ROUTE_ID.Home)).toBe('/')
    })
  })

  describe(ROUTE_ID.Session, () => {
    test.each<{ params: { sID: string }; expected: string }>([
      { params: { sID: 'abc' }, expected: '/s/abc' },
      { params: { sID: 'foo/bar' }, expected: '/s/foo%2Fbar' },
    ])('$expected', ({ params, expected }) => {
      expect(pathTo(ROUTE_ID.Session, params)).toBe(expected)
    })
  })

  describe(ROUTE_ID.Request, () => {
    test.each<{ params: { sID: string; rID: string }; expected: string }>([
      { params: { sID: 'abc', rID: 'def' }, expected: '/s/abc/def' },
      { params: { sID: 'abc', rID: 'x/y' }, expected: '/s/abc/x%2Fy' },
    ])('$expected', ({ params, expected }) => {
      expect(pathTo(ROUTE_ID.Request, params)).toBe(expected)
    })
  })
})

describe('createRoutes', () => {
  const children = createRoutes()[0]?.children ?? []

  test.each([
    { id: ROUTE_ID.Home, index: true },
    { id: ROUTE_ID.Session, path: 's/:sID' },
  ])('route $id is registered in root children', ({ id, ...shape }) => {
    expect(children.find((r) => r.id === id)).toMatchObject(shape)
  })

  test('request route is nested under session', () => {
    const sessionRoute = children.find((r) => r.id === ROUTE_ID.Session)
    expect(sessionRoute?.children?.find((r) => r.id === ROUTE_ID.Request)).toMatchObject({ path: ':rID' })
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

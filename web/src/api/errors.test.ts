import { test, expect } from 'vitest'
import { APIErrorCommon, APIErrorNotFound, APIErrorUnknown } from './errors'

test('errors', () => {
  expect(new APIErrorNotFound().description.toLowerCase()).contains('not found')
  expect(new APIErrorCommon().description.toLowerCase()).contains('server')
  expect(new APIErrorUnknown().description.toLowerCase()).contains("don't know")
})

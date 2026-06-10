import { APIErrorBadRequest, APIErrorCommon, APIErrorNotFound } from './errors'
import type { APIError } from './errors'
import { describe, expect, test } from 'vitest'

describe('API error classes', () => {
  const ERROR_CLASSES = [
    {
      Class: APIErrorCommon,
      description: "Something went wrong on the server side, but we can't identify it as a specific error",
    },
    {
      Class: APIErrorBadRequest,
      description: 'Bad request',
    },
    {
      Class: APIErrorNotFound,
      description: 'Not found',
    },
  ] as const

  test.each(ERROR_CLASSES)('$Class.name is an instance of Error', ({ Class }) => {
    expect(new Class()).toBeInstanceOf(Error)
  })

  test.each(ERROR_CLASSES)('$Class.name satisfies APIError interface', ({ Class }) => {
    const error: APIError = new Class()
    expect(error).toHaveProperty('description')
    expect(error).toHaveProperty('message')
  })

  test.each(ERROR_CLASSES)('$Class.name has correct description', ({ Class, description }) => {
    expect(new Class().description).toBe(description)
  })

  test.each(ERROR_CLASSES)('$Class.name sets message when provided', ({ Class }) => {
    const message = 'Custom error message'
    expect(new Class({ message }).message).toBe(message)
  })

  test.each(ERROR_CLASSES)('$Class.name has empty message when not provided', ({ Class }) => {
    expect(new Class().message).toBe('')
  })

  test.each(ERROR_CLASSES)('$Class.name sets response when provided', ({ Class }) => {
    const response = new Response(null, { status: 400 })
    expect(new Class({ response }).response).toBe(response)
  })

  test.each(ERROR_CLASSES)('$Class.name has undefined response when not provided', ({ Class }) => {
    expect(new Class().response).toBeUndefined()
  })

  test.each(ERROR_CLASSES)('$Class.name description is readonly (does not change across instances)', ({ Class }) => {
    const a = new Class()
    const b = new Class()
    expect(a.description).toBe(b.description)
  })
})

import { APIErrorBadRequest, APIErrorNotFound } from './errors'
import { extractErrorMessage } from './utils'
import type { Middleware } from 'openapi-fetch'

/** Middleware to throw an APIErrorBadRequest if the response status is 400. */
export const throwIfBadRequest: Middleware = {
  async onResponse({ response: r }): Promise<undefined> {
    if (r.status === 400) {
      throw new APIErrorBadRequest({
        message: (await extractErrorMessage(r)) || 'Bad request (400)',
        response: r,
      })
    }
  },
}

/** Middleware to throw an APIErrorNotFound if the response status is 404. */
export const throwIfNotFound: Middleware = {
  async onResponse({ response: r }): Promise<undefined> {
    if (r.status === 404) {
      throw new APIErrorNotFound({
        message: (await extractErrorMessage(r)) || 'Not found (404)',
        response: r,
      })
    }
  },
}

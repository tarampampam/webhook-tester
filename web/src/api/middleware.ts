import type { Middleware } from 'openapi-fetch'
import { APIErrorCommon, APIErrorNotFound } from './errors'

/** This middleware throws an error if the response is not JSON. */
export const throwIfNotJSON: Middleware = {
  async onResponse({ response }): Promise<undefined> {
    // check the response header "Content-Type" to be sure it's JSON
    if (response.headers.get('content-type')?.toLowerCase().includes('json')) {
      return undefined
    }

    throw new APIErrorCommon({
      message: 'Response is not JSON (the header "Content-Type" does not include "json")',
      response: response,
    })
  },
}

/** This middleware throws a well-formatted error if the response is not OK. */
export const throwIfNotValidResponse: Middleware = {
  async onResponse({ response }): Promise<undefined> {
    // skip if the response is OK
    if (response.ok) {
      return undefined
    }

    let message: string = response.statusText

    // try to parse the response body as JSON and extract the error message
    if (response.headers.get('content-type')?.toLowerCase().includes('json')) {
      try {
        const data = await response.clone().json()

        switch (true) {
          case data.message && typeof data.message === 'string': // { message: "..." }
            message = data.message
            break

          case data.error && typeof data.error === 'string': // { error: "..." }
            message = data.error
            break

          case data.errors && Array.isArray(data.errors) && data.errors.length > 0: // { errors: ["...", ...] }
            message = data.errors.filter((e: unknown) => typeof e === 'string').join(', ')
            break

          case data.errors && typeof data.errors === 'object': // { errors: { "...": "..." } }
            message = Object.values(data.errors as Record<string, unknown>)
              .filter((e) => typeof e === 'string')
              .join(', ')
            break
        }
      } catch (e) {
        message += ' (failed to parse the response body as JSON)'
      }
    }

    // handle some common HTTP status codes
    switch (response.status) {
      case 404:
        throw new APIErrorNotFound({ message, response: response })
    }

    throw new APIErrorCommon({ message, response: response })
  },
}

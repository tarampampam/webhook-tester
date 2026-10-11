/** Extracts a meaningful error message from a Response object. */
export const extractErrorMessage = async (r: Response): Promise<string | null> => {
  if (r.ok) {
    return null
  }

  const contentType = r.headers.get('content-type')?.toLowerCase()

  if (contentType) {
    switch (true) {
      case contentType.includes('json'): {
        let data: unknown = undefined

        try {
          data = await r.clone().json()
        } catch (e: unknown) {
          throw new Error('Failed to parse the response body as JSON: ' + (e instanceof Error ? e.message : String(e)))
        }

        switch (typeof data) {
          case 'string':
            return data

          case 'object': {
            if (data === null) {
              return null
            }

            if (Array.isArray(data) && data.length === 0) {
              return null
            }

            if (Object.keys(data).length === 0) {
              return null
            }

            const obj = data as Record<string, unknown>

            switch (true) {
              case !!(obj.message && typeof obj.message === 'string'): // { message: "..." }
                return obj.message satisfies string

              case !!(obj.error && typeof obj.error === 'string'): // { error: "..." }
                return obj.error satisfies string

              case !!(obj.errors && Array.isArray(obj.errors)): // { errors: [...] }
                if ((obj.errors as unknown[]).length === 0) {
                  return null
                }

                return (obj.errors as unknown[]).filter((e: unknown) => typeof e === 'string').join(', ')

              case !!(obj.errors && typeof obj.errors === 'object'): // { errors: { "...": "..." } }
                return Object.values(obj.errors as Record<string, unknown>)
                  .filter((e) => typeof e === 'string')
                  .join(', ')

              default:
                try {
                  return JSON.stringify(data)
                } catch {
                  return null
                }
            }
          }

          default:
            return null
        }
      }

      case contentType.includes('text/'):
        return r.clone().text()

      default:
        return null // unsupported content type
    }
  }

  return null // Content-Type header is missing
}

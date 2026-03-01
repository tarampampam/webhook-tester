import createClient, { type Client as OpenapiClient, type ClientOptions } from 'openapi-fetch'
import { coerce as semverCoerce, parse as semverParse, type SemVer } from 'semver'
import { base64ToUint8Array, uint8ArrayToBase64 } from '~/shared'
import { APIErrorUnknown } from './errors'
import { throwIfNotJSON, throwIfNotValidResponse } from './middleware'
import { components, paths, type RequestEventAction } from './schema.gen'

type AppSettings = Readonly<{
  limits: Readonly<{
    maxRequests: number
    maxRequestBodySize: number // In bytes
    sessionTTL: number // In seconds
  }>
  tunnel: Readonly<{
    enabled: boolean
    url: URL | null
  }>
  publicUrlRoot: URL | null
}>

type SessionOptions = Readonly<{
  uuid: string
  response: Readonly<{
    statusCode: number
    headers: ReadonlyArray<{ name: string; value: string }>
    delay: number
    body: Readonly<Uint8Array>
  }>
  createdAt: Readonly<Date>
}>

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS' | 'CONNECT' | 'TRACE' | string

type CapturedRequest = Readonly<{
  uuid: string
  clientAddress: string
  method: HttpMethod
  requestPayload: Uint8Array
  headers: ReadonlyArray<{ name: string; value: string }>
  url: Readonly<URL>
  capturedAt: Readonly<Date>
}>

type RequestEvent = Readonly<{
  action: RequestEventAction
  request: {
    uuid: string
    clientAddress: string
    method: HttpMethod
    headers: ReadonlyArray<{ name: string; value: string }>
    url: Readonly<URL>
    capturedAt: Readonly<Date>
  } | null
}>

export class Client {
  private readonly baseUrl: URL
  private readonly api: OpenapiClient<paths>
  private cache: Partial<{
    currentVersion: Readonly<SemVer>
    latestVersion: Readonly<SemVer>
    settings: AppSettings
  }> = {}

  constructor(opt?: ClientOptions) {
    const baseUrl: string | null = opt?.baseUrl
      ? opt.baseUrl
      : typeof window !== 'undefined' // for non-browser environments, like tests
        ? window.location.protocol + '//' + window.location.host
        : null

    if (!baseUrl) {
      throw new Error('The base URL is not provided and cannot be determined')
    }

    this.baseUrl = new URL(baseUrl)

    this.api = createClient<paths>({ ...opt, baseUrl: baseUrl.toString() })
    this.api.use(throwIfNotJSON, throwIfNotValidResponse)
  }

  /**
   * Returns the version of the app.
   *
   * @throws {APIError}
   */
  async currentVersion(force: boolean = false): Promise<Readonly<SemVer>> {
    if (this.cache.currentVersion && !force) {
      return this.cache.currentVersion
    }

    const { data, response } = await this.api.GET('/api/version', { priority: 'low' })

    if (data) {
      const version = semverParse(semverCoerce(data.version.replace('@', '-')))

      if (!version) {
        throw new APIErrorUnknown({ message: `Failed to parse the current version value: ${data.version}`, response })
      }

      this.cache.currentVersion = Object.freeze(version)

      return this.cache.currentVersion
    }

    throw new APIErrorUnknown({ message: response.statusText, response }) // will never happen due to the middleware
  }

  /**
   * Returns the latest available version of the app.
   *
   * @throws {APIError}
   */
  async latestVersion(force: boolean = false): Promise<Readonly<SemVer>> {
    if (this.cache.latestVersion && !force) {
      return this.cache.latestVersion
    }

    const { data, response } = await this.api.GET('/api/version/latest', { priority: 'low' })

    if (data) {
      const version = semverParse(semverCoerce(data.version))

      if (!version) {
        throw new APIErrorUnknown({ message: `Failed to parse the latest version value: ${data.version}`, response })
      }

      this.cache.latestVersion = Object.freeze(version)

      return this.cache.latestVersion
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Returns the app settings.
   *
   * @throws {APIError}
   */
  async getSettings(force: boolean = false): Promise<AppSettings> {
    if (this.cache.settings && !force) {
      return this.cache.settings
    }

    const { data, response } = await this.api.GET('/api/settings')

    if (data) {
      this.cache.settings = Object.freeze({
        limits: Object.freeze({
          maxRequests: data.limits.max_requests,
          maxRequestBodySize: data.limits.max_request_body_size,
          sessionTTL: data.limits.session_ttl, // in seconds
        }),
        tunnel: Object.freeze({
          enabled: data.tunnel.enabled,
          url: data?.tunnel.url ? new URL(data.tunnel.url) : null,
        }),
        publicUrlRoot: data?.public_url_root ? new URL(data.public_url_root) : null,
      })

      return this.cache.settings
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Creates a new session with the specified response settings.
   *
   * @throws {APIError}
   */
  async newSession({
    statusCode = 200,
    headers = {},
    delay = 0,
    responseBody = new Uint8Array(),
  }: {
    statusCode?: number
    headers?: Record<string, string>
    delay?: number
    responseBody?: Uint8Array
  }): Promise<SessionOptions> {
    const { data, response } = await this.api.POST('/api/session', {
      body: {
        status_code: Math.min(Math.max(100, statusCode), 530), // clamp to the valid range
        headers: Object.entries(headers)
          .map(([name, value]) => ({ name, value })) // convert to array of objects
          .filter((h) => h.value), // remove empty values
        delay: Math.min(Math.max(0, delay), 30), // clamp to the valid range
        response_body_base64: uint8ArrayToBase64(responseBody),
      },
    })

    if (data) {
      return Object.freeze({
        uuid: data.uuid,
        response: Object.freeze({
          statusCode: data.response.status_code,
          headers: Array.from(data.response.headers).map(({ name, value }) => Object.freeze({ name, value })),
          delay: data.response.delay,
          body: base64ToUint8Array(data.response.response_body_base64),
        }),
        createdAt: Object.freeze(new Date(data.created_at_unix_milli)),
      })
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Returns the session by its ID.
   *
   * @throws {APIError}
   */
  async getSession(sID: string): Promise<SessionOptions> {
    const { data, response } = await this.api.GET(`/api/session/{session_uuid}`, {
      params: { path: { session_uuid: sID } },
    })

    if (data) {
      return Object.freeze({
        uuid: data.uuid,
        response: Object.freeze({
          statusCode: data.response.status_code,
          headers: Array.from(data.response.headers).map(({ name, value }) => Object.freeze({ name, value })),
          delay: data.response.delay,
          body: base64ToUint8Array(data.response.response_body_base64),
        }),
        createdAt: Object.freeze(new Date(data.created_at_unix_milli)),
      })
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Batch checking the existence of the sessions by their IDs.
   *
   * @throws {APIError}
   */
  async checkSessionExists<T extends string>(...ids: Array<T>): Promise<{ [K in T]: boolean }> {
    const { data, response } = await this.api.POST('/api/session/check/exists', {
      body: ids,
    })

    if (data) {
      // first, create an object with keys from the input array and values as `false`
      const result = Object.fromEntries(ids.map((id) => [id, false])) as { [K in T]: boolean }

      // next, iterate over the response data and set the value to `true` if the ID exists and is `true`
      for (const id in data) {
        if (data[id] === true) {
          result[id as T] = true
        }
      }

      return Object.freeze(result)
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Deletes the session by its ID.
   *
   * @throws {APIError}
   */
  async deleteSession(sID: string): Promise<boolean> {
    const { data, response } = await this.api.DELETE('/api/session/{session_uuid}', {
      params: { path: { session_uuid: sID } },
    })

    if (data) {
      return data.success
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Returns the list of captured requests for the session by its ID.
   *
   * @throws {APIError}
   */
  async getSessionRequests(sID: string): Promise<ReadonlyArray<CapturedRequest>> {
    const { data, response } = await this.api.GET('/api/session/{session_uuid}/requests', {
      params: { path: { session_uuid: sID } },
    })

    if (data) {
      return Object.freeze(
        Array.from(data)
          // convert the list of requests to the immutable objects with the correct types
          .map((req) =>
            Object.freeze({
              uuid: req.uuid,
              clientAddress: req.client_address,
              method: req.method,
              requestPayload: base64ToUint8Array(req.request_payload_base64),
              headers: Object.freeze(Array.from(req.headers).map(({ name, value }) => Object.freeze({ name, value }))),
              url: Object.freeze(new URL(req.url)),
              capturedAt: Object.freeze(new Date(req.captured_at_unix_milli)),
            })
          )
          // sort the list by capturedAt date, to have the latest requests first
          .sort((a, b) => b.capturedAt.getTime() - a.capturedAt.getTime())
      )
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Deletes all captured requests for the session by its ID.
   *
   * @throws {APIError}
   */
  async deleteAllSessionRequests(sID: string): Promise<boolean> {
    const { data, response } = await this.api.DELETE('/api/session/{session_uuid}/requests', {
      params: { path: { session_uuid: sID } },
    })

    if (data) {
      return data.success
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Subscribes to the captured requests for the session by its ID.
   *
   * The promise resolves with a closer function that can be called to close the WebSocket connection.
   */
  async subscribeToSessionRequests(
    sID: string,
    {
      onConnected,
      onUpdate,
      onError,
    }: {
      onConnected?: () => void // called when the WebSocket connection is established
      onUpdate: (request: RequestEvent) => void // called when the update is received
      onError?: (err: Error) => void // called when an error occurs on alive connection
    }
  ): Promise</* closer */ () => void> {
    const protocol = this.baseUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    const path: keyof paths = '/api/session/{session_uuid}/requests/subscribe'

    return new Promise((resolve: (closer: () => void) => void, reject: (err: Error) => void) => {
      let connected: boolean = false

      try {
        const ws = new WebSocket(`${protocol}//${this.baseUrl.host}${path.replace('{session_uuid}', sID)}`)

        ws.onopen = (): void => {
          connected = true
          onConnected?.()
          resolve((): void => ws.close())
        }

        ws.onerror = (event: Event): void => {
          // convert Event to Error
          const err = new Error(event instanceof ErrorEvent ? String(event.error) : 'WebSocket error')

          if (connected) {
            onError?.(err)
          }

          reject(err) // will be ignored if the promise is already resolved
        }

        ws.onmessage = (event): void => {
          if (event.data) {
            const req = JSON.parse(event.data) as components['schemas']['RequestEvent']
            const payload: RequestEvent = {
              action: req.action,
              request: req.request
                ? Object.freeze({
                    uuid: req.request.uuid,
                    clientAddress: req.request.client_address,
                    method: req.request.method,
                    headers: Object.freeze(req.request.headers),
                    url: Object.freeze(new URL(req.request.url)),
                    capturedAt: Object.freeze(new Date(req.request.captured_at_unix_milli)),
                  })
                : null,
            }

            onUpdate(Object.freeze(payload))
          }
        }
      } catch (e) {
        // convert any exception to Error
        const err = e instanceof Error ? e : new Error(String(e))

        if (connected) {
          onError?.(err)
        }

        reject(err)
      }
    })
  }

  /**
   * Returns the captured request by its ID.
   *
   * @throws {APIError}
   */
  async getSessionRequest(sID: string, rID: string): Promise<CapturedRequest> {
    const { data, response } = await this.api.GET('/api/session/{session_uuid}/requests/{request_uuid}', {
      params: { path: { session_uuid: sID, request_uuid: rID } },
    })

    if (data) {
      return Object.freeze({
        uuid: data.uuid,
        clientAddress: data.client_address,
        method: data.method,
        requestPayload: base64ToUint8Array(data.request_payload_base64),
        headers: Object.freeze(Array.from(data.headers)),
        url: Object.freeze(new URL(data.url)),
        capturedAt: Object.freeze(new Date(data.captured_at_unix_milli)),
      })
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }

  /**
   * Deletes the captured request by its ID.
   *
   * @throws {APIError}
   */
  async deleteSessionRequest(sID: string, rID: string): Promise<boolean> {
    const { data, response } = await this.api.DELETE('/api/session/{session_uuid}/requests/{request_uuid}', {
      params: { path: { session_uuid: sID, request_uuid: rID } },
    })

    if (data) {
      return data.success
    }

    throw new APIErrorUnknown({ message: response.statusText, response })
  }
}

import createClient, { type Client as OpenapiClient, type ClientOptions } from 'openapi-fetch'
import { base64ToUint8Array, uint8ArrayToBase64 } from '~/shared'
import { APIErrorCommon } from './errors'
import { throwIfBadRequest, throwIfNotFound } from './middleware'
import type { components, paths, RequestEventAction } from './schema.gen'

type RequestOptions = {
  priority?: RequestPriority
  signal?: AbortSignal
}

export class Client {
  private readonly baseUrl: URL
  private readonly api: OpenapiClient<paths>

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
    this.api.use(throwIfBadRequest)
    this.api.use(throwIfNotFound)
  }

  /**
   * Returns the version of the app.
   *
   * @throws {APIError}
   */
  async currentVersion(opts?: RequestOptions): Promise<Readonly<string>> {
    const { data, response: r } = await this.api.GET('/api/version', {
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze(data.version.replace(/^[vV]/, '')) // remove leading "v" from version
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Returns the latest available version of the app.
   *
   * @throws {APIError}
   */
  async latestVersion(opts?: RequestOptions): Promise<Readonly<string>> {
    const { data, response: r } = await this.api.GET('/api/version/latest', {
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze(data.version.replace(/^[vV]/, '')) // remove leading "v" from version
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Returns the app settings.
   *
   * @throws {APIError}
   */
  async getSettings(opts?: RequestOptions): Promise<AppSettings> {
    const { data, response: r } = await this.api.GET('/api/settings', {
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze({
        limits: Object.freeze({
          maxRequests: data.limits.max_requests,
          maxRequestBodySize: data.limits.max_request_body_size,
          sessionTTL: data.limits.session_ttl, // in seconds
        }),
        tunnel: Object.freeze({
          enabled: data.tunnel.enabled,
          url: data?.tunnel.url ? new URL(data.tunnel.url) : null,
        }),
        publicUrlRoot: data?.public_url_root ? new URL(data.public_url_root.replace(/\/+$/, '')) : null,
      })
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Creates a new session with the specified response settings.
   *
   * @throws {APIError}
   */
  async newSession(
    {
      statusCode = 200,
      headers = {},
      delay = 0,
      responseBody = new Uint8Array(),
    }: {
      statusCode?: number
      headers?: Record<string, string>
      delay?: number
      responseBody?: Uint8Array
    },
    opts?: RequestOptions
  ): Promise<
    Readonly<{
      uuid: string
      createdAt: Readonly<Date>
    }>
  > {
    const { data, response: r } = await this.api.POST('/api/session', {
      body: {
        status_code: Math.min(Math.max(100, statusCode), 530), // clamp to the valid range
        headers: Object.entries(headers)
          .map(([name, value]) => ({ name, value })) // convert to array of objects
          .filter((h) => h.value), // remove empty values
        delay: Math.min(Math.max(0, delay), 30), // clamp to the valid range
        response_body_base64: uint8ArrayToBase64(responseBody),
      },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze({
        uuid: data.uuid,
        createdAt: Object.freeze(new Date(data.created_at_unix_milli)),
      })
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Returns the session by its ID.
   *
   * @throws {APIError}
   */
  async getSession(sID: string, opts?: RequestOptions): Promise<SessionOptions> {
    const { data, response: r } = await this.api.GET('/api/session/{session_uuid}', {
      params: { path: { session_uuid: sID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze({
        uuid: data.uuid,
        response: Object.freeze({
          statusCode: data.response.status_code,
          headers: Object.freeze(
            Array.from(data.response.headers).map(({ name, value }) => Object.freeze({ name, value }))
          ),
          delay: data.response.delay,
          body: base64ToUint8Array(data.response.response_body_base64),
        }),
        createdAt: Object.freeze(new Date(data.created_at_unix_milli)),
      })
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Batch checking the existence of the sessions by their IDs.
   *
   * @throws {APIError}
   */
  async checkSessionExists<T extends string>(ids: Array<T>, opts?: RequestOptions): Promise<{ [K in T]: boolean }> {
    const { data, response: r } = await this.api.POST('/api/session/check/exists', {
      body: ids,
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
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

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Deletes the session by its ID.
   *
   * @throws {APIError}
   */
  async deleteSession(sID: string, opts?: RequestOptions): Promise<boolean> {
    const { data, response: r } = await this.api.DELETE('/api/session/{session_uuid}', {
      params: { path: { session_uuid: sID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return data.success
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Returns the list of captured requests for the session by its ID.
   *
   * @throws {APIError}
   */
  async getSessionRequests(sID: string, opts?: RequestOptions): Promise<ReadonlyArray<CapturedRequest>> {
    const { data, response: r } = await this.api.GET('/api/session/{session_uuid}/requests', {
      params: { path: { session_uuid: sID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze(
        Array.from(data)
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
          .sort((a, b) => b.capturedAt.getTime() - a.capturedAt.getTime())
      )
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Deletes all captured requests for the session by its ID.
   *
   * @throws {APIError}
   */
  async deleteAllSessionRequests(sID: string, opts?: RequestOptions): Promise<boolean> {
    const { data, response: r } = await this.api.DELETE('/api/session/{session_uuid}/requests', {
      params: { path: { session_uuid: sID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return data.success
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Subscribes to the captured requests for the session by its ID.
   *
   * The promise resolves with a closer function that can be called to close the WebSocket connection.
   */
  async subscribeToSessionRequests(
    sID: string,
    handlers: {
      onConnected?: () => void // called when the WebSocket connection is established
      onUpdate: (request: RequestEvent) => void // called when the update is received
      onError?: (err: Error) => void // called when an error occurs on alive connection
    },
    opts?: { signal?: AbortSignal }
  ): Promise<() => void> {
    opts?.signal?.throwIfAborted()

    const protocol = this.baseUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    const path: keyof paths = '/api/session/{session_uuid}/requests/subscribe'

    return new Promise<() => void>((resolve, reject: (err: Error) => void) => {
      let connected: boolean = false

      try {
        const ws = new WebSocket(`${protocol}//${this.baseUrl.host}${path.replace('{session_uuid}', sID)}`)

        const makeAbortError = (): Error => {
          const reason = opts?.signal?.reason
          return reason instanceof Error ? reason : new DOMException('The operation was aborted', 'AbortError')
        }

        const onAbort = (): void => ws.close(4000, 'Aborted by signal')
        const cleanup = (): void => opts?.signal?.removeEventListener('abort', onAbort)

        opts?.signal?.addEventListener('abort', onAbort, { once: true })

        ws.onopen = (): void => {
          connected = true
          handlers.onConnected?.()
          resolve((): void => {
            cleanup()
            ws.close(1000, 'Closed by caller')
          })
        }

        ws.onerror = (event: Event): void => {
          cleanup()

          const err = opts?.signal?.aborted
            ? makeAbortError()
            : new Error(event instanceof ErrorEvent ? String(event.error) : 'WebSocket error')

          if (connected) {
            handlers.onError?.(err)
          }

          reject(err)
        }

        ws.onclose = (): void => {
          cleanup()

          if (!connected) {
            reject(
              opts?.signal?.aborted
                ? makeAbortError()
                : new Error('WebSocket connection closed before it was established')
            )
            return
          }

          if (opts?.signal?.aborted) {
            handlers.onError?.(makeAbortError())
          }
        }

        ws.onmessage = (event): void => {
          if (!event.data) {
            return
          }

          try {
            const req = JSON.parse(event.data) as components['schemas']['RequestEvent']
            const payload: RequestEvent = {
              action: req.action,
              request: req.request
                ? Object.freeze({
                    uuid: req.request.uuid,
                    clientAddress: req.request.client_address,
                    method: req.request.method,
                    headers: Object.freeze(
                      Array.from(req.request.headers).map(({ name, value }) => Object.freeze({ name, value }))
                    ),
                    url: Object.freeze(new URL(req.request.url)),
                    capturedAt: Object.freeze(new Date(req.request.captured_at_unix_milli)),
                  })
                : null,
            }

            handlers.onUpdate(Object.freeze(payload))
          } catch (e) {
            handlers.onError?.(e instanceof Error ? e : new Error(String(e)))
          }
        }
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e))

        if (connected) {
          handlers.onError?.(err)
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
  async getSessionRequest(sID: string, rID: string, opts?: RequestOptions): Promise<CapturedRequest> {
    const { data, response: r } = await this.api.GET('/api/session/{session_uuid}/requests/{request_uuid}', {
      params: { path: { session_uuid: sID, request_uuid: rID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return Object.freeze({
        uuid: data.uuid,
        clientAddress: data.client_address,
        method: data.method,
        requestPayload: base64ToUint8Array(data.request_payload_base64),
        headers: Object.freeze(Array.from(data.headers).map(({ name, value }) => Object.freeze({ name, value }))),
        url: Object.freeze(new URL(data.url)),
        capturedAt: Object.freeze(new Date(data.captured_at_unix_milli)),
      })
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }

  /**
   * Deletes the captured request by its ID.
   *
   * @throws {APIError}
   */
  async deleteSessionRequest(sID: string, rID: string, opts?: RequestOptions): Promise<boolean> {
    const { data, response: r } = await this.api.DELETE('/api/session/{session_uuid}/requests/{request_uuid}', {
      params: { path: { session_uuid: sID, request_uuid: rID } },
      priority: opts?.priority,
      signal: opts?.signal,
    })

    if (r.ok && data) {
      return data.success
    }

    throw new APIErrorCommon({
      message: r.statusText,
      response: r,
    })
  }
}

type AppSettings = Readonly<{
  limits: Readonly<{
    maxRequests: number
    maxRequestBodySize: number // in bytes
    sessionTTL: number // in seconds
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

import {Fetcher} from 'openapi-typescript-fetch'
import {paths, components} from './schema'
import {Base64} from 'js-base64'

const fetcher = Fetcher.for<paths>()
const textEncoder = new TextEncoder()
const textDecoder = new TextDecoder('utf-8')

export function getAppVersion(): Promise<string> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/version').method('get').create()
      .call(fetcher, {})
      .then((resp) => resolve(resp.data.version))
      .catch(reject)
  })
}

export interface APISettingsResponse {
  limits: {
    maxRequests: number
    maxWebhookBodySize: number
    sessionLifetimeSec: number
  }
}

export function getAppSettings(): Promise<APISettingsResponse> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/settings').method('get').create()
      .call(fetcher, {})
      .then((resp) => resolve({
        limits: {
          maxRequests: resp.data.limits.max_requests,
          maxWebhookBodySize: resp.data.limits.max_webhook_body_size,
          sessionLifetimeSec: resp.data.limits.session_lifetime_sec,
        }
      }))
      .catch(reject)
  })
}

export interface APINewSessionRequest {
  statusCode?: number
  contentType?: string
  responseDelay?: number
  responseContent?: Uint8Array
}

export interface APINewSessionResponse {
  UUID: string
  response: {
    content: Uint8Array
    code: Number
    contentType: string
    delaySec: Number
  }
  createdAt: Date
}

export function startNewSession(request: APINewSessionRequest): Promise<APINewSessionResponse> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session').method('post').create()
      .call(fetcher, {
        content_type: request.contentType,
        response_content_base64: Base64.encode(textDecoder.decode(request.responseContent)),
        response_delay: request.responseDelay,
        status_code: request.statusCode,
      })
      .then((resp) => resolve({
        UUID: resp.data.uuid,
        response: {
          content: textEncoder.encode(Base64.decode(resp.data.response.content_base64)),
          code: resp.data.response.code,
          contentType: resp.data.response.content_type,
          delaySec: resp.data.response.delay_sec,
        },
        createdAt: new Date(resp.data.created_at_unix * 1000),
      }))
      .catch(reject)
  })
}

export function deleteSession(sessionUUID: string): Promise<boolean> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session/{session_uuid}').method('delete').create()
      .call(fetcher, {session_uuid: sessionUUID})
      .then((resp) => resolve(resp.data.success))
      .catch(reject)
  })
}

export type HTTPMethod = 'GET' | 'HEAD' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'OPTIONS' | 'TRACE'

export interface RecordedRequest {
  UUID: string
  clientAddress: string
  method: HTTPMethod
  content: Uint8Array
  headers: { name: string, value: string }[]
  url: string // relative (`/foo/bar`, NOT `http://example.com/foo/bar`)
  createdAt: Date
}

export function getSessionRequest(sessionUUID: string, requestUUID: string): Promise<RecordedRequest> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session/{session_uuid}/requests/{request_uuid}').method('get').create()
      .call(fetcher, {session_uuid: sessionUUID, request_uuid: requestUUID})
      .then((resp) => resolve({
        UUID: resp.data.uuid,
        clientAddress: resp.data.client_address,
        method: resp.data.method,
        content: textEncoder.encode(Base64.decode(resp.data.content_base64)),
        headers: resp.data.headers.map(h => ({name: h.name, value: h.value})),
        url: resp.data.url,
        createdAt: new Date(resp.data.created_at_unix * 1000),
      }))
      .catch(reject)
  })
}

export function getAllSessionRequests(sessionUUID: string): Promise<RecordedRequest[]> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session/{session_uuid}/requests').method('get').create()
      .call(fetcher, {session_uuid: sessionUUID})
      .then((resp) => resolve(resp.data.map((req: components['schemas']['SessionRequest']) => ({
        UUID: req.uuid,
        clientAddress: req.client_address,
        method: req.method,
        content: textEncoder.encode(Base64.decode(req.content_base64)),
        headers: req.headers.map(h => ({name: h.name, value: h.value})),
        url: req.url,
        createdAt: new Date(req.created_at_unix * 1000),
      }))))
      .catch(reject)
  })
}

export function deleteSessionRequest(sessionUUID: string, requestUUID: string): Promise<boolean> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session/{session_uuid}/requests/{request_uuid}').method('delete').create()
      .call(fetcher, {session_uuid: sessionUUID, request_uuid: requestUUID})
      .then((resp) => resolve(resp.data.success))
      .catch(reject)
  })
}

export function deleteAllSessionRequests(sessionUUID: string): Promise<boolean> {
  return new Promise((resolve, reject) => {
    fetcher.path('/api/session/{session_uuid}/requests').method('delete').create()
      .call(fetcher, {session_uuid: sessionUUID})
      .then((resp) => resolve(resp.data.success))
      .catch(reject)
  })
}

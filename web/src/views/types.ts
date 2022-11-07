export interface NewSessionSettings {
  statusCode?: number
  responseDelay?: number
  contentType?: string
  responseContent?: Uint8Array
  destroyCurrentSession?: boolean
}

export interface RecordedRequest {
  UUID: string
  clientAddress: string
  method: 'get' | 'head' | 'post' | 'put' | 'patch' | 'delete' | 'connect' | 'options' | 'trace' | 'unknown'
  content: Uint8Array
  headers: { name: string, value: string }[]
  url: string // relative (`/foo/bar`, NOT `http://example.com/foo/bar`)
  createdAt: Date
}

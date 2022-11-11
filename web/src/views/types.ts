export interface NewSessionSettings {
  statusCode?: number
  responseDelay?: number
  contentType?: string
  responseContent?: Uint8Array
  destroyCurrentSession?: boolean
}

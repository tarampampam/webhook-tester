export interface RecordedRequest {
    uuid: string
    client_address: string
    method: string
    when: Date
    content: string
    headers: {
        name: string
        value: string
    }[]
    url: string // relative (`/foo/bar`, NOT `http://example.com/foo/bar`)
}

export interface NewSessionData {
    statusCode: number | null
    contentType: string | null
    responseDelay: number | null
    responseBody: string | null
    destroyCurrentSession: boolean
}

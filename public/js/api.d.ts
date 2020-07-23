/**
 * This interfaces is used only for correct IDE type-hinting.
 */

export interface APISettings {
    version: string
    limits: {
        session_lifetime_sec: number
        max_requests: number
    }
}

export interface APIDeleteSession {
    success: boolean
}

export interface APINewSession {
    uuid: string
    response: {
        content: string
        code: number
        content_type: string
        delay_sec: number
        created_at_unix: number
    }
}

export interface APINewSessionSettings {
    status_code: string | null
    content_type: string | null
    response_delay: string | null
    response_body: string | null
}

export interface APIRecordedRequest {
    client_address: string
    method: string
    content: string
    headers: {
        [key: string]: string;
    }
    url: string // relative (`/foo/bar`, NOT `http://example.com/foo/bar`)
    created_at_unix: number
}

export interface APIDeleteSessionRequest {
    success: boolean
}

export interface APIDeleteAllSessionRequests {
    success: boolean
}

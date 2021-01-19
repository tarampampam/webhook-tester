package storage

// WebHookResponse is a sever webhook response settings.
type WebHookResponse struct {
	Content     string // Static server content
	Code        uint16 // Default server response code
	ContentType string // Response type (basically it used as `Content-Type` header value)
	DelaySec    uint8  // Server delay before server response sending
}

// SessionData describes session settings (like response data and any additional information).
type SessionData struct {
	UUID            string          // Unique session identifier
	WebHookResponse WebHookResponse // WebHook response settings
	CreatedAtUnix   int64           // When session was created
}

// Request is a recorded request information.
type Request struct {
	ClientAddr string            // Client hostname or IP address (who sent this request)
	Method     string            // HTTP method name (eg.: 'GET', 'POST')
	Content    string            // Request body (payload)
	Headers    map[string]string // HTTP headers
	URI        string            // Requested URI
}

// RequestData describes recorded request and additional meta-data.
type RequestData struct {
	UUID          string  // Unique request identifier
	Request       Request // Request data
	CreatedAtUnix int64   // When request was created
}

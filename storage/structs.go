package storage

// Response is a sever webhook response settings.
type WebHookResponse struct {
	// Static server content
	Content string

	// Default server response code
	Code uint16

	// Response type (basically it used as `Content-Type` header value)
	ContentType string

	// Server delay before server response sending
	DelaySec uint8
}

// SessionData describes session settings (like response data and any additional information).
type SessionData struct {
	// Unique session identifier
	UUID string

	// WebHook response settings
	WebHookResponse WebHookResponse

	// When session was created
	CreatedAtUnix int64
}

// Request is a recorded request information.
type Request struct {
	// Client hostname or IP address (who sent this request)
	ClientAddr string

	// HTTP method name (eg.: 'GET', 'POST')
	Method string

	// Request body (payload)
	Content string

	// HTTP headers
	Headers map[string]string

	// Requested URI
	URI string
}

// RequestData describes recorded request and additional meta-data.
type RequestData struct {
	// Unique request identifier
	UUID string

	// Request data
	Request Request

	// When request was created
	CreatedAtUnix int64
}

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

	// When this session was created
	CreatedAtUnix int64
}

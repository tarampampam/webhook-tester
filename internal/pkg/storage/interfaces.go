package storage

type Storage interface {
	// GetSession returns session data.
	// If session was not found - `nil, nil` will be returned.
	GetSession(sessionUUID string) (*SessionData, error)

	// CreateSession creates new session in storage using passed data.
	CreateSession(webHookSettings *WebHookResponse) (*SessionData, error)

	// DeleteSession deletes session with passed UUID.
	DeleteSession(sessionUUID string) (bool, error)

	// DeleteRequests deletes stored requests for session with passed UUID.
	DeleteRequests(sessionUUID string) (bool, error)

	// CreateRequest creates new request in storage using passed data.
	// Session with passed UUID must exists.
	CreateRequest(sessionUUID string, r *Request) (*RequestData, error)

	// GetRequest returns request data.
	// If request was not found - `nil, nil` will be returned.
	GetRequest(sessionUUID, requestUUID string) (*RequestData, error)

	// GetAllRequests returns all request as a slice of structures.
	// If requests was not found - `nil, nil` will be returned.
	GetAllRequests(sessionUUID string) (*[]RequestData, error)

	// DeleteRequest deletes stored request with passed session and request UUIDs.
	DeleteRequest(sessionUUID, requestUUID string) (bool, error)
}

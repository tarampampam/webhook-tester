package storage

import (
	"time"

	"github.com/google/uuid"
)

// Storage is a Session's and Request's storage.
type Storage interface {
	// GetSession returns session data.
	// If session was not found - `nil, nil` will be returned.
	GetSession(uuid string) (Session, error)

	// CreateSession creates new session in storage using passed data.
	// Session UUID without error will be returned on success.
	CreateSession(content []byte, code uint16, contentType string, delay time.Duration, id ...string) (string, error)

	// DeleteSession deletes session with passed UUID.
	DeleteSession(uuid string) (bool, error)

	// DeleteRequests deletes stored requests for session with passed UUID.
	DeleteRequests(uuid string) (bool, error)

	// CreateRequest creates new request in storage using passed data and updates expiration time for session and all
	// stored requests for the session.
	// Session with passed UUID must exist.
	// Request UUID without error will be returned on success.
	CreateRequest(sessionUUID, clientAddr, method, uri string, content []byte, headers map[string]string) (string, error)

	// GetRequest returns request data.
	// If request was not found - `nil, nil` will be returned.
	GetRequest(sessionUUID, requestUUID string) (Request, error)

	// GetAllRequests returns all request as a slice of structures.
	// If requests was not found - `nil, nil` will be returned.
	GetAllRequests(sessionUUID string) ([]Request, error)

	// DeleteRequest deletes stored request with passed session and request UUIDs.
	DeleteRequest(sessionUUID, requestUUID string) (bool, error)
}

// Session describes session settings (like response data and any additional information).
type Session interface {
	UUID() string         // UUID returns unique session identifier.
	Content() []byte      // Content returns session server response content.
	Code() uint16         // Code returns default server response code.
	ContentType() string  // ContentType returns response content type.
	Delay() time.Duration // Delay returns delay before response sending.
	CreatedAt() time.Time // CreatedAt returns creation time (accuracy to seconds).
}

// Request describes recorded request and additional meta-data.
type Request interface {
	UUID() string               // UUID returns unique request identifier.
	ClientAddr() string         // ClientAddr returns client hostname or IP address (who sent this request).
	Method() string             // Method returns HTTP method name (eg.: 'GET', 'POST').
	Content() []byte            // Content returns request body (payload).
	Headers() map[string]string // Headers returns HTTP request headers.
	URI() string                // URI returns Uniform Resource Identifier.
	CreatedAt() time.Time       // CreatedAt returns creation time (accuracy to seconds).
}

// NewUUID generates new UUID v4.
func NewUUID() string { return uuid.New().String() }

// IsValidUUID checks if passed string is valid UUID v4.
func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)

	return err == nil
}

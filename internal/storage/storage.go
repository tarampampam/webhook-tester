package storage

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrSessionNotFound = fmt.Errorf("session %w", ErrNotFound)
	ErrRequestNotFound = fmt.Errorf("request %w", ErrNotFound)

	ErrClosed = errors.New("closed")
)

// Storage manages Session and Request data.
type Storage interface {
	// NewSession creates a new session and returns a session UUID on success.
	// The Session.CreatedAt field will be set to the current time.
	NewSession(Session) (sID string, _ error)

	// GetSession retrieves session data.
	// If the session is not found, ErrSessionNotFound will be returned.
	GetSession(sID string) (*Session, error)

	// DeleteSession removes the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteSession(sID string) error

	// NewRequest creates a new request for the session with the specified UUID and returns a request UUID on success.
	// The session with the specified UUID must exist. The Request.CreatedAt field will be set to the current time.
	// If the session is not found, ErrSessionNotFound will be returned.
	NewRequest(sID string, _ Request) (rID string, _ error)

	// GetRequest retrieves request data.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	GetRequest(sID, rID string) (*Request, error)

	// GetAllRequests returns all requests for the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned. If there are no requests, an empty map
	// will be returned.
	GetAllRequests(sID string) (map[string]Request, error)

	// DeleteRequest removes the request with the specified UUID.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	DeleteRequest(sID, rID string) error

	// DeleteAllRequests removes all requests for the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteAllRequests(sID string) error
}

type (
	// Session describes session settings (like response data and any additional information).
	Session struct {
		Code        uint16        `msgpack:"code" json:"code"`                 // default server response code
		Content     []byte        `msgpack:"content" json:"content"`           // session server response content
		ContentType string        `msgpack:"content_type" json:"content_type"` // response content type
		Delay       time.Duration `msgpack:"delay" json:"delay"`               // delay before response sending
		CreatedAt   time.Time     `msgpack:"created_at" json:"created_at"`     // creation time (accuracy to milliseconds)
	}

	// Request describes recorded request and additional meta-data.
	Request struct {
		ClientAddr string            `msgpack:"client_addr" json:"client_addr"` // client hostname or IP address
		Method     string            `msgpack:"method" json:"method"`           // HTTP method name (i.e., 'GET', 'POST')
		Body       []byte            `msgpack:"body" json:"body"`               // request body (payload)
		Headers    map[string]string `msgpack:"headers" json:"headers"`         // HTTP request headers
		URL        url.URL           `msgpack:"url" json:"url"`                 // Uniform Resource Identifier
		CreatedAt  time.Time         `msgpack:"created_at" json:"created_at"`   // creation time (accuracy to milliseconds)
	}
)

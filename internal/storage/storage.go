package storage

import (
	"context"
	"errors"
	"fmt"
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
	// NewSession creates a new session and returns a session ID on success.
	// The Session.CreatedAt field will be set to the current time.
	NewSession(_ context.Context, _ Session, id ...string) (sID string, _ error)

	// GetSession retrieves session data.
	// If the session is not found, ErrSessionNotFound will be returned.
	GetSession(_ context.Context, sID string) (*Session, error)

	// AddSessionTTL adds the specified TTL to the session (and all its requests) with the specified ID.
	AddSessionTTL(_ context.Context, sID string, howMuch time.Duration) error

	// DeleteSession removes the session with the specified ID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteSession(_ context.Context, sID string) error

	// NewRequest creates a new request for the session with the specified ID and returns a request ID on success.
	// The session with the specified ID must exist. The Request.CreatedAtUnixMilli field will be set to the
	// current time. The storage may limit the number of requests per session - in this case the oldest request
	// will be removed.
	// If the session is not found, ErrSessionNotFound will be returned.
	NewRequest(_ context.Context, sID string, _ Request) (rID string, _ error)

	// GetRequest retrieves request data.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	GetRequest(_ context.Context, sID, rID string) (*Request, error)

	// GetAllRequests returns all requests for the session with the specified ID.
	// If the session is not found, ErrSessionNotFound will be returned. If there are no requests, an empty map
	// will be returned.
	GetAllRequests(_ context.Context, sID string) (map[string]Request, error)

	// DeleteRequest removes the request with the specified ID.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	DeleteRequest(_ context.Context, sID, rID string) error

	// DeleteAllRequests removes all requests for the session with the specified ID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteAllRequests(_ context.Context, sID string) error
}

type (
	// Session describes session settings (like response data and any additional information).
	Session struct {
		Code               uint16        `json:"code"`                  // default server response code
		Headers            []HttpHeader  `json:"headers"`               // server response headers
		ResponseBody       []byte        `json:"body"`                  // server response body (payload)
		Delay              time.Duration `json:"delay"`                 // delay before response sending
		CreatedAtUnixMilli int64         `json:"created_at_unit_milli"` // creation time
		ExpiresAt          time.Time     `json:"-"`                     // expiration time (doesn't store in the storage)
	}

	// Request describes recorded request and additional meta-data.
	Request struct {
		ClientAddr         string       `json:"client_addr"`           // client hostname or IP address
		Method             string       `json:"method"`                // HTTP method name (i.e., 'GET', 'POST')
		Body               []byte       `json:"body"`                  // request body (payload)
		Headers            []HttpHeader `json:"headers"`               // HTTP request headers
		URL                string       `json:"url"`                   // Uniform Resource Identifier
		CreatedAtUnixMilli int64        `json:"created_at_unit_milli"` // creation time
	}

	HttpHeader struct {
		Name  string `json:"name"`  // the name of the header, e.g., "Content-Type"
		Value string `json:"value"` // the value of the header, e.g., "application/json"
	}
)

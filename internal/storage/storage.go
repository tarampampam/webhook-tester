package storage

import (
	"context"
	"encoding/json"
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
	NewSession(context.Context, Session) (sID string, _ error)

	// GetSession retrieves session data.
	// If the session is not found, ErrSessionNotFound will be returned.
	GetSession(_ context.Context, sID string) (*Session, error)

	// DeleteSession removes the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteSession(_ context.Context, sID string) error

	// NewRequest creates a new request for the session with the specified UUID and returns a request UUID on success.
	// The session with the specified UUID must exist. The Request.CreatedAt field will be set to the current time.
	// The storage may limit the number of requests per session - in this case the oldest request will be removed.
	// If the session is not found, ErrSessionNotFound will be returned.
	NewRequest(_ context.Context, sID string, _ Request) (rID string, _ error)

	// GetRequest retrieves request data.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	GetRequest(_ context.Context, sID, rID string) (*Request, error)

	// GetAllRequests returns all requests for the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned. If there are no requests, an empty map
	// will be returned.
	GetAllRequests(_ context.Context, sID string) (map[string]Request, error)

	// DeleteRequest removes the request with the specified UUID.
	// If the request or session is not found, ErrNotFound (ErrSessionNotFound or ErrRequestNotFound) will be returned.
	DeleteRequest(_ context.Context, sID, rID string) error

	// DeleteAllRequests removes all requests for the session with the specified UUID.
	// If the session is not found, ErrSessionNotFound will be returned.
	DeleteAllRequests(_ context.Context, sID string) error
}

type (
	// Session describes session settings (like response data and any additional information).
	Session struct {
		Code        uint16        `json:"code"`         // default server response code
		Content     []byte        `json:"content"`      // session server response content
		ContentType string        `json:"content_type"` // response content type
		Delay       time.Duration `json:"delay"`        // delay before response sending
		CreatedAt   Time          `json:"created_at"`   // creation time
	}

	// Request describes recorded request and additional meta-data.
	Request struct {
		ClientAddr string            `json:"client_addr"` // client hostname or IP address
		Method     string            `json:"method"`      // HTTP method name (i.e., 'GET', 'POST')
		Body       []byte            `json:"body"`        // request body (payload)
		Headers    map[string]string `mjson:"headers"`    // HTTP request headers
		URL        URL               `json:"url"`         // Uniform Resource Identifier
		CreatedAt  Time              `json:"created_at"`  // creation time
	}
)

// --------------------------------------------------------------------------------------------------------------------

// Time is a custom time.Time type that marshals and unmarshals time in unix-nano format.
type Time struct{ time.Time }

var (
	_ json.Marshaler   = (*Time)(nil) // ensure that Time implements the json.Marshaler interface
	_ json.Unmarshaler = (*Time)(nil) // ensure that Time implements the json.Unmarshaler interface
)

// MarshalJSON implements the json.Marshaler interface and returns the time in unix-nano format.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("0"), nil
	}

	return []byte(fmt.Sprintf("%d", t.Time.UnixNano())), nil // fmt.Sprintf used here to avoid exponential notation
}

// UnmarshalJSON implements the json.Unmarshaler interface and parses the time in unix-nano format.
func (t *Time) UnmarshalJSON(data []byte) error {
	var unixNano int64
	if err := json.Unmarshal(data, &unixNano); err != nil {
		return err
	}

	if unixNano == 0 {
		t.Time = time.Time{}

		return nil
	}

	t.Time = time.Unix(0, unixNano)

	return nil
}

// --------------------------------------------------------------------------------------------------------------------

// URL is a custom url.URL type that marshals and unmarshals URL as a string.
type URL struct{ url.URL }

var (
	_ json.Marshaler   = (*URL)(nil) // ensure that URL implements the json.Marshaler interface
	_ json.Unmarshaler = (*URL)(nil) // ensure that URL implements the json.Unmarshaler interface
)

// MarshalJSON implements the json.Marshaler interface and returns the URL as a string.
func (u URL) MarshalJSON() ([]byte, error) { return json.Marshal(u.String()) }

// UnmarshalJSON implements the json.Unmarshaler interface and parses the URL as a string.
func (u *URL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return err
	}

	u.URL = *parsed

	return nil
}

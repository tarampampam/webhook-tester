// Package storage provides abstractions and implementations for persisting webhook sessions
// and captured HTTP requests.
package storage

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"
)

var (
	// ErrNotFound is the base sentinel for missing-resource errors. The specific variants
	// ([ErrSessionNotFound], [ErrRequestNotFound]) wrap it, so [errors.Is](err, ErrNotFound) matches all of them.
	ErrNotFound = errors.New("not found")
	// ErrSessionNotFound is returned when a session does not exist or has expired.
	ErrSessionNotFound = fmt.Errorf("session %w", ErrNotFound)
	// ErrRequestNotFound is returned when a captured request does not exist.
	ErrRequestNotFound = fmt.Errorf("request %w", ErrNotFound)

	// ErrAlreadyExists is the base sentinel for duplicate-resource errors. The specific variants
	// ([ErrSessionAlreadyExists], [ErrRequestAlreadyExists]) wrap it.
	ErrAlreadyExists = errors.New("already exists")
	// ErrSessionAlreadyExists is returned when creating a session whose ID is already in use.
	ErrSessionAlreadyExists = fmt.Errorf("session %w", ErrAlreadyExists)
	// ErrRequestAlreadyExists is returned when creating a request whose ID is already in use within a session.
	ErrRequestAlreadyExists = fmt.Errorf("request %w", ErrAlreadyExists)

	// ErrClosed is returned by any operation called after Close has been invoked on the storage.
	ErrClosed = errors.New("storage closed")
)

type (
	// ResponseHeader is a single HTTP header name-value pair included in the session's response.
	ResponseHeader struct {
		Name  string
		Value string
	}

	// SessionResponse defines the HTTP response the storage returns to every incoming webhook request.
	SessionResponse struct {
		Code    uint16
		Headers []ResponseHeader
		Body    []byte
		Delay   time.Duration // artificial delay before the response is sent; zero means no delay
	}

	// SessionMeta holds server-assigned session metadata. Callers receive it from the storage;
	// they cannot supply these values on creation.
	SessionMeta struct {
		CreatedAt time.Time
		ExpiresAt time.Time // zero value means the session has no expiration
	}

	// Session is the complete session record: its configured response and server-assigned metadata.
	Session struct {
		Response SessionResponse
		Meta     SessionMeta
	}
)

// NoExpiration disables session expiration when passed as the ttl argument to NewSession or AddSessionTTL.
const NoExpiration time.Duration = 0

// SessionStorage manages session data, including response configuration and metadata.
type SessionStorage interface {
	// NewSession creates a session with the given ID and response configuration.
	// Pass [NoExpiration] as ttl to create a session that never expires.
	// Returns [ErrSessionAlreadyExists] if a live session with that ID already exists.
	// The returned [SessionMeta] carries the server-assigned timestamps.
	NewSession(_ context.Context, sID string, _ SessionResponse, ttl time.Duration) (*SessionMeta, error)

	// GetSession returns the session with the given ID.
	// Returns [ErrSessionNotFound] if the session does not exist or has expired.
	GetSession(_ context.Context, sID string) (*Session, error)

	// AddSessionTTL sets the session expiry to now+howMuch and applies the same deadline to all
	// captured requests. Pass [NoExpiration] to remove the expiry entirely.
	// Returns [ErrSessionNotFound] if the session does not exist or has expired.
	AddSessionTTL(_ context.Context, sID string, howMuch time.Duration) error

	// DeleteSession removes the session and all its captured requests.
	// Returns [ErrSessionNotFound] if the session does not exist.
	DeleteSession(_ context.Context, sID string) error
}

type (
	// RequestHeader is a single HTTP header name-value pair from a captured webhook request.
	RequestHeader struct {
		Name  string
		Value string
	}

	// CapturedRequest holds the raw data from an incoming HTTP request to a webhook endpoint.
	CapturedRequest struct {
		ClientAddr string
		Method     string
		Body       []byte
		Headers    []RequestHeader
		URL        string
	}

	// RequestMeta holds server-assigned metadata for a captured request.
	RequestMeta struct {
		CreatedAt time.Time
	}

	// Request is the complete record of a captured webhook request.
	Request struct {
		Data CapturedRequest
		Meta RequestMeta
	}
)

// RequestStorage manages captured webhook requests, which are associated with sessions.
type RequestStorage interface {
	// NewRequest records a captured request under sID/rID.
	// When the per-session request cap is reached, the oldest request is evicted to make room.
	// Returns [ErrSessionNotFound] if the session does not exist or has expired.
	// Returns [ErrRequestAlreadyExists] if a request with rID already exists in the session.
	// The returned [RequestMeta] carries the server-assigned creation timestamp.
	NewRequest(_ context.Context, sID, rID string, _ CapturedRequest) (*RequestMeta, error)

	// GetRequest returns the captured request identified by sID and rID.
	// Returns [ErrSessionNotFound] or [ErrRequestNotFound] if either does not exist.
	GetRequest(_ context.Context, sID, rID string) (*Request, error)

	// GetRequests returns an iterator over all captured requests for the session, ordered newest-first.
	// The returned error covers setup failures, including [ErrSessionNotFound].
	// Errors that occur mid-iteration are written to *err if err is non-nil; always check it after
	// the loop, not before. Passing nil disables mid-iteration error reporting.
	GetRequests(_ context.Context, sID string, err *error) (iter.Seq2[string, Request], error)

	// DeleteRequest removes the captured request identified by sID and rID.
	// Returns [ErrSessionNotFound] or [ErrRequestNotFound] if either does not exist.
	DeleteRequest(_ context.Context, sID, rID string) error

	// DeleteAllRequests removes all captured requests for the given session.
	// Returns [ErrSessionNotFound] if the session does not exist.
	DeleteAllRequests(_ context.Context, sID string) error
}

// Storage combines session and request management.
type Storage interface {
	SessionStorage
	RequestStorage
}

// --------------------------------------------------------------------------------------------------------------------

// sortEntry is a lightweight sort key used when building ordered snapshots for iteration.
type sortEntry struct {
	id        string
	createdAt time.Time
}

package storage

import "time"

type Storage interface {
	// Close closes connections to the storage.
	Close() error

	// CreateSession creates new session in storage using passed options.
	CreateSession(webHookSettings *WebHookResponse, ttl time.Duration) (*SessionData, error)

	// DeleteSession deletes session with passed UUID.
	DeleteSession(sessionUUID string) (bool, error)

	// DeleteRequests deletes stored requests for session with passed UUID.
	DeleteRequests(sessionUUID string) (bool, error)
}

package storage

import "time"

type Storage interface {
	// Close closes connections to the storage.
	Close() error

	// NewSession creates new session in storage using passed options.
	NewSession(webHookSettings *WebHookResponse, ttl time.Duration) (*SessionData, error)
}

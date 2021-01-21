package storage

import "time"

type InMemoryStorage struct {
	sessionTTL  time.Duration
	maxRequests uint16

	cleanupInterval time.Duration


}

func NewInMemoryStorage(sessionTTL time.Duration, maxRequests uint16, cleanupInterval time.Duration) *InMemoryStorage {
	return &InMemoryStorage{sessionTTL: sessionTTL, maxRequests: maxRequests, cleanupInterval: cleanupInterval}
}

// GetSession returns session data.
func (s *InMemoryStorage) GetSession(uuid string) (Session, error) {
	return nil, nil // TODO implement
}

// CreateSession creates new session in storage using passed data.
func (s *InMemoryStorage) CreateSession(content string, code uint16, contentType string, delay time.Duration) (string, error) { //nolint:lll
	return "", nil // TODO implement
}

// DeleteSession deletes session with passed UUID.
func (s *InMemoryStorage) DeleteSession(uuid string) (bool, error) {
	return false, nil // TODO implement
}

// DeleteRequests deletes stored requests for session with passed UUID.
func (s *InMemoryStorage) DeleteRequests(uuid string) (bool, error) {
	return false, nil // TODO implement
}

// CreateRequest creates new request in storage using passed data.
func (s *InMemoryStorage) CreateRequest(sessionUUID, clientAddr, method, content, uri string, headers map[string]string) (string, error) { //nolint:lll
	return "", nil // TODO implement
}

// GetRequest returns request data.
func (s *InMemoryStorage) GetRequest(sessionUUID, requestUUID string) (Request, error) {
	return nil, nil // TODO implement
}

// GetAllRequests returns all request as a slice of structures.
func (s *InMemoryStorage) GetAllRequests(sessionUUID string) ([]Request, error) {
	return nil, nil // TODO implement
}

// DeleteRequest deletes stored request with passed session and request UUIDs.
func (s *InMemoryStorage) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	return false, nil // TODO implement
}

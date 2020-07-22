package http

import (
	"time"
	"webhook-tester/storage"
)

type fakeStorage struct{}

func (*fakeStorage) Close() error                                    { return nil }
func (*fakeStorage) DeleteSession(sessionUUID string) (bool, error)  { return true, nil }
func (*fakeStorage) DeleteRequests(sessionUUID string) (bool, error) { return true, nil }
func (*fakeStorage) CreateSession(wh *storage.WebHookResponse, ttl time.Duration) (*storage.SessionData, error) {
	return &storage.SessionData{}, nil
}

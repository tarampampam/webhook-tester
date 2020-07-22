package http

import (
	"time"
	"webhook-tester/storage"
)

type fakeStorage struct{}

func (*fakeStorage) Close() error { return nil }
func (*fakeStorage) NewSession(wh *storage.WebHookResponse, ttl time.Duration) (*storage.SessionData, error) {
	return &storage.SessionData{}, nil
}

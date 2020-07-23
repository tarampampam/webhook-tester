package http

import (
	"webhook-tester/storage"
)

type fakeStorage struct{}

func (*fakeStorage) Close() error                                    { return nil }
func (*fakeStorage) DeleteSession(sessionUUID string) (bool, error)  { return true, nil }
func (*fakeStorage) DeleteRequests(sessionUUID string) (bool, error) { return true, nil }
func (*fakeStorage) GetSession(sessionUUID string) (*storage.SessionData, error) {
	return &storage.SessionData{}, nil
}
func (*fakeStorage) CreateSession(wh *storage.WebHookResponse) (*storage.SessionData, error) {
	return &storage.SessionData{}, nil
}
func (*fakeStorage) CreateRequest(sessionUUID string, r *storage.Request) (*storage.RequestData, error) {
	return &storage.RequestData{}, nil
}
func (*fakeStorage) GetRequest(sessionUUID, requestUUID string) (*storage.RequestData, error) {
	return &storage.RequestData{}, nil
}
func (*fakeStorage) GetAllRequests(sessionUUID string) (*[]storage.RequestData, error) {
	return new([]storage.RequestData), nil
}

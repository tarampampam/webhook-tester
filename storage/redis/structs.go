package redis

import "webhook-tester/storage"

type sessionData struct {
	ResponseContent     string `json:"resp_content"`
	ResponseCode        uint16 `json:"resp_code"`
	ResponseContentType string `json:"resp_content_type"`
	ResponseDelaySec    uint8  `json:"resp_delay_sec"`
	CreatedAtUnix       int64  `json:"created_at_unix"`
}

func (s *sessionData) toSharedStruct(sessionUUID string) *storage.SessionData {
	return &storage.SessionData{
		UUID: sessionUUID,
		WebHookResponse: storage.WebHookResponse{
			Content:     s.ResponseContent,
			Code:        s.ResponseCode,
			ContentType: s.ResponseContentType,
			DelaySec:    s.ResponseDelaySec,
		},
		CreatedAtUnix: s.CreatedAtUnix,
	}
}

type requestData struct {
	ClientAddr    string            `json:"client_addr"`
	Method        string            `json:"method"`
	Content       string            `json:"content"`
	Headers       map[string]string `json:"headers"`
	URI           string            `json:"uri"`
	CreatedAtUnix int64             `json:"created_at_unix"`
}

func (r *requestData) toSharedStruct(requestUUID string) *storage.RequestData {
	return &storage.RequestData{
		UUID: requestUUID,
		Request: storage.Request{
			ClientAddr: r.ClientAddr,
			Method:     r.Method,
			Content:    r.Content,
			Headers:    r.Headers,
			URI:        r.URI,
		},
		CreatedAtUnix: r.CreatedAtUnix,
	}
}

type storageKey struct {
	sessionUUID string
}

func newStorageKey(sessionUUID string) storageKey {
	return storageKey{sessionUUID: sessionUUID}
}

// session data [session-UUID]:[session-data]
func (s storageKey) session() string {
	return "webhook-tester:session:" + s.sessionUUID
}

// requests list [timestamp]:[request-UUID]
func (s storageKey) requests() string {
	return s.session() + ":requests"
}

// request data [request-UUID]:[request-data]
func (s storageKey) request(requestUUID string) string {
	return s.session() + ":requests:" + requestUUID
}

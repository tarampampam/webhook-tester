package redis

import (
	"context"
	"time"
	"webhook-tester/storage"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

const sessionKeyPrefix string = "session:"
const sessionRequestsKeyPrefix string = "requests:"

type Storage struct {
	Context       context.Context
	client        *redis.Client
	json          jsoniter.API
	uuidGenerator func() string
}

type sessionData struct {
	ResponseContent     string `json:"resp_content"`
	ResponseCode        uint16 `json:"resp_code"`
	ResponseContentType string `json:"resp_content_type"`
	ResponseDelaySec    uint8  `json:"resp_delay_sec"`
	CreatedAtUnix       int64  `json:"created_at_unix"`
}

type requestData struct {
	ClientAddr    string            `json:"client_addr"`
	Method        string            `json:"method"`
	Content       string            `json:"content"`
	Headers       map[string]string `json:"headers"`
	URI           string            `json:"uri"`
	CreatedAtUnix int64             `json:"created_at_unix"`
}

func NewStorage(addr, password string, dbNum, maxConn int) *Storage {
	return &Storage{
		Context: context.Background(),
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: "",
			Password: password,
			DB:       dbNum,
			PoolSize: maxConn,
		}),
		json: jsoniter.ConfigFastest,
		uuidGenerator: func() string {
			return uuid.New().String()
		},
	}
}

func (s *Storage) Close() error {
	return s.client.Close()
}

func (s *Storage) CreateSession(wh *storage.WebHookResponse, ttl time.Duration) (*storage.SessionData, error) {
	var (
		sessionUUID = s.uuidGenerator()
		sessionData = sessionData{
			ResponseContent:     wh.Content,
			ResponseCode:        wh.Code,
			ResponseContentType: wh.ContentType,
			ResponseDelaySec:    wh.DelaySec,
			CreatedAtUnix:       time.Now().Unix(),
		}
	)

	asJSON, jsonErr := s.json.Marshal(sessionData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	if err := s.client.Set(s.Context, sessionKeyPrefix+sessionUUID, asJSON, ttl).Err(); err != nil {
		return nil, err
	}

	return &storage.SessionData{
		UUID: sessionUUID,
		WebHookResponse: storage.WebHookResponse{
			Content:     sessionData.ResponseContent,
			Code:        sessionData.ResponseCode,
			ContentType: sessionData.ResponseContentType,
			DelaySec:    sessionData.ResponseDelaySec,
		},
		CreatedAtUnix: sessionData.CreatedAtUnix,
	}, nil
}

func (s *Storage) deleteKeys(keys ...string) (bool, error) {
	cmdResult := s.client.Del(s.Context, keys...)

	if err := cmdResult.Err(); err != nil {
		return false, err
	}

	if count, err := cmdResult.Result(); err != nil {
		return false, err
	} else if count == 0 {
		return false, nil
	}

	return true, nil
}

func (s *Storage) DeleteSession(sessionUUID string) (bool, error) {
	return s.deleteKeys(sessionKeyPrefix + sessionUUID)
}

func (s *Storage) DeleteRequests(sessionUUID string) (bool, error) {
	return s.deleteKeys(sessionRequestsKeyPrefix + sessionUUID)
}

func (s *Storage) CreateRequest(sessionUUID string, r *storage.Request) (*storage.RequestData, error) {
	var (
		requestUUID = s.uuidGenerator()
		requestData = requestData{
			ClientAddr:    r.ClientAddr,
			Method:        r.Method,
			Content:       r.Content,
			Headers:       r.Headers,
			URI:           r.URI,
			CreatedAtUnix: time.Now().Unix(),
		}
	)

	asJSON, jsonErr := s.json.Marshal(requestData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	if err := s.client.HSet(s.Context, sessionRequestsKeyPrefix+sessionUUID, requestUUID, asJSON).Err(); err != nil {
		return nil, err
	}

	// @todo: append? ttl?

	return &storage.RequestData{
		UUID: requestUUID,
		Request: storage.Request{
			ClientAddr: requestData.ClientAddr,
			Method:     requestData.Method,
			Content:    requestData.Content,
			Headers:    requestData.Headers,
			URI:        requestData.URI,
		},
		CreatedAtUnix: requestData.CreatedAtUnix,
	}, nil
}

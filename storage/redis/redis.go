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

type Storage struct {
	ctx           context.Context
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

func NewStorage(addr, password string, dbNum, maxConn int) *Storage {
	return &Storage{
		ctx: context.Background(),
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

func (s *Storage) NewSession(wh *storage.WebHookResponse, ttl time.Duration) (*storage.SessionData, error) {
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

	if err := s.client.Set(s.ctx, sessionKeyPrefix+sessionUUID, asJSON, ttl).Err(); err != nil {
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

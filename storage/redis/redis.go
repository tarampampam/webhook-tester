package redis

import (
	"context"
	"time"
	"webhook-tester/storage"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

type Storage struct {
	Context       context.Context
	ttl           time.Duration
	maxRequests   uint16
	client        *redis.Client
	json          jsoniter.API
	uuidGenerator func() string
}

func NewStorage(addr, password string, dbNum, maxConn int, sessionTTL time.Duration, maxRequests uint16) *Storage {
	return &Storage{
		Context:     context.Background(),
		ttl:         sessionTTL,
		maxRequests: maxRequests,
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

func (s *Storage) GetSession(sessionUUID string) (*storage.SessionData, error) {
	value, err := s.client.Get(s.Context, newStorageKey(sessionUUID).session()).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	sessionData := sessionData{}
	if err := s.json.Unmarshal(value, &sessionData); err != nil {
		return nil, err
	}

	return sessionData.toSharedStruct(sessionUUID), nil
}

func (s *Storage) CreateSession(wh *storage.WebHookResponse) (*storage.SessionData, error) {
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

	if err := s.client.Set(s.Context, newStorageKey(sessionUUID).session(), asJSON, s.ttl).Err(); err != nil {
		return nil, err
	}

	return sessionData.toSharedStruct(sessionUUID), nil
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
	return s.deleteKeys(newStorageKey(sessionUUID).session())
}

func (s *Storage) DeleteRequests(sessionUUID string) (bool, error) {
	key := newStorageKey(sessionUUID)

	return s.deleteKeys(key.requestsOrderKey(), key.requestsDataKey())
}

func (s *Storage) syncRequests(key storageKey) error {
	// get keys from ordered requests set (key is request UUID)
	orderedKeys, ordErr := s.client.LRange(s.Context, key.requestsOrderKey(), 0, int64(s.maxRequests)-1).Result()
	if ordErr != nil {
		return ordErr
	}

	// get all requests from data key (key also is request UUID)
	requestKeys, reqErr := s.client.HKeys(s.Context, key.requestsDataKey()).Result()
	if reqErr != nil {
		return reqErr
	}

	// calculate difference between ordered keys set and data key
	orderedMap := make(map[string]bool, len(orderedKeys))
	for _, key := range orderedKeys {
		orderedMap[key] = true
	}

	var diff []string

	for _, key := range requestKeys {
		if _, found := orderedMap[key]; !found {
			diff = append(diff, key) // only if key exists in data key and not exists in ordered
		}
	}

	if len(diff) > 0 {
		// and remove them
		if err := s.client.HDel(s.Context, key.requestsDataKey(), diff...).Err(); err != nil {
			return err
		}
	}

	return nil
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
		key = newStorageKey(sessionUUID)
	)

	asJSON, jsonErr := s.json.Marshal(requestData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	// execute pipeline <https://redis.io/topics/pipelining>
	if _, err := s.client.Pipelined(s.Context, func(pipe redis.Pipeliner) error {
		// save request uuid into ordered requests list
		pipe.LPush(s.Context, key.requestsOrderKey(), requestUUID)
		// trim requests list
		pipe.LTrim(s.Context, key.requestsOrderKey(), 0, int64(s.maxRequests)-1)
		// save request data
		pipe.HSet(s.Context, key.requestsDataKey(), requestUUID, asJSON)

		// update ttl for required keys
		pipe.Expire(s.Context, key.requestsOrderKey(), s.ttl)
		pipe.Expire(s.Context, key.requestsDataKey(), s.ttl)
		pipe.Expire(s.Context, key.session(), s.ttl)

		return nil
	}); err != nil {
		return nil, err
	}

	if err := s.syncRequests(key); err != nil {
		return nil, err
	}

	return requestData.toSharedStruct(requestUUID), nil
}

func (s *Storage) GetRequest(sessionUUID, requestUUID string) (*storage.RequestData, error) {
	value, err := s.client.HGet(s.Context, newStorageKey(sessionUUID).requestsDataKey(), requestUUID).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	requestData := requestData{}
	if err := s.json.Unmarshal(value, &requestData); err != nil {
		return nil, err
	}

	return requestData.toSharedStruct(requestUUID), nil
}

func (s *Storage) GetAllRequests(sessionUUID string) (*[]storage.RequestData, error) {
	data, err := s.client.HGetAll(s.Context, newStorageKey(sessionUUID).requestsDataKey()).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	result := make([]storage.RequestData, 0)

	for requestUUID, json := range data {
		requestData := requestData{}
		if err := s.json.Unmarshal([]byte(json), &requestData); err == nil { // ignore errors with wrong json
			result = append(result, *requestData.toSharedStruct(requestUUID))
		}
	}

	return &result, nil
}

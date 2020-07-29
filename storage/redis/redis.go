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
	redis         *redis.Client
	json          jsoniter.API
	uuidGenerator func() string
}

func NewStorage(addr, password string, dbNum, maxConn int, sessionTTL time.Duration, maxRequests uint16) *Storage {
	return &Storage{
		Context:     context.TODO(),
		ttl:         sessionTTL,
		maxRequests: maxRequests,
		redis: redis.NewClient(&redis.Options{
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
	return s.redis.Close()
}

func (s *Storage) GetSession(sessionUUID string) (*storage.SessionData, error) {
	value, err := s.redis.Get(s.Context, newStorageKey(sessionUUID).session()).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	var sessionData = sessionData{}

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

	if err := s.redis.Set(s.Context, newStorageKey(sessionUUID).session(), asJSON, s.ttl).Err(); err != nil {
		return nil, err
	}

	return sessionData.toSharedStruct(sessionUUID), nil
}

func (s *Storage) deleteKeys(keys ...string) (bool, error) {
	cmdResult := s.redis.Del(s.Context, keys...)

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

	// get request UUIDs, associated with session
	requestUUIDs, readErr := s.redis.ZRangeByScore(s.Context, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if readErr != nil {
		return false, readErr
	}

	// removing plan
	var keys = []string{key.requests()}
	for _, requestUUID := range requestUUIDs {
		keys = append(keys, key.request(requestUUID))
	}

	return s.deleteKeys(keys...)
}

func (s *Storage) CreateRequest(sessionUUID string, r *storage.Request) (*storage.RequestData, error) { //nolint:funlen
	var (
		requestUUID = s.uuidGenerator()
		now         = time.Now()
		requestData = requestData{
			ClientAddr:    r.ClientAddr,
			Method:        r.Method,
			Content:       r.Content,
			Headers:       r.Headers,
			URI:           r.URI,
			CreatedAtUnix: now.Unix(),
		}
		key = newStorageKey(sessionUUID)
	)

	asJSON, jsonErr := s.json.Marshal(requestData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	// save request data
	if _, err := s.redis.Pipelined(s.Context, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(s.Context, key.requests(), &redis.Z{
			Score:  float64(now.UnixNano()),
			Member: requestUUID,
		})
		pipe.Set(s.Context, key.request(requestUUID), asJSON, s.ttl)

		return nil
	}); err != nil {
		return nil, err
	}

	// read all stored request UUIDs
	requestUUIDs, readErr := s.redis.ZRangeByScore(s.Context, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if readErr != nil {
		return nil, readErr
	}

	// if currently we have more than allowed requests - remove unnecessary
	if len(requestUUIDs) > int(s.maxRequests) {
		if _, err := s.redis.Pipelined(s.Context, func(pipe redis.Pipeliner) error {
			for _, k := range requestUUIDs[:len(requestUUIDs)-int(s.maxRequests)] {
				pipe.ZRem(s.Context, key.requests(), k)
				pipe.Del(s.Context, key.request(k))
			}

			return nil
		}); err != nil {
			return nil, err
		}
	}

	// update expiring date
	if _, err := s.redis.Pipelined(s.Context, func(pipe redis.Pipeliner) error {
		if len(requestUUIDs) > 0 {
			forUpdate := make([]string, 0)

			if len(requestUUIDs) > int(s.maxRequests) {
				forUpdate = requestUUIDs[len(requestUUIDs)-int(s.maxRequests):]
			} else {
				forUpdate = append(forUpdate, requestUUIDs...)
			}

			for _, k := range forUpdate {
				pipe.Expire(s.Context, key.request(k), s.ttl)
			}
		}
		pipe.Expire(s.Context, key.requests(), s.ttl)
		pipe.Expire(s.Context, key.session(), s.ttl)

		return nil
	}); err != nil {
		return nil, err
	}

	return requestData.toSharedStruct(requestUUID), nil
}

func (s *Storage) GetRequest(sessionUUID, requestUUID string) (*storage.RequestData, error) {
	value, err := s.redis.Get(s.Context, newStorageKey(sessionUUID).request(requestUUID)).Bytes()

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
	var key = newStorageKey(sessionUUID)

	UUIDs, allErr := s.redis.ZRangeByScore(s.Context, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if allErr != nil {
		if allErr == redis.Nil {
			return nil, nil // not found
		}

		return nil, allErr
	}

	result := make([]storage.RequestData, 0)

	if len(UUIDs) > 0 {
		// convert request UUIDs into storage keys
		keys := make([]string, len(UUIDs))
		for i, UUID := range UUIDs {
			keys[i] = key.request(UUID)
		}

		// read all requests in a one request
		rawRequests, gettingErr := s.redis.MGet(s.Context, keys...).Result()
		if gettingErr != nil {
			return nil, gettingErr
		}

		// convert response into a map, where key is request UUID and data is a raw json string
		requestsMap := make(map[string]string, len(UUIDs))
		for i, UUID := range UUIDs { //nolint:wsl
			if asJSON, ok := rawRequests[i].(string); ok {
				requestsMap[UUID] = asJSON
			}
		}

		// convert responses map into result response
		for UUID, asJSON := range requestsMap {
			requestData := requestData{}
			if err := s.json.Unmarshal([]byte(asJSON), &requestData); err == nil { // ignore errors with wrong json
				result = append(result, *requestData.toSharedStruct(UUID))
			}
		}
	}

	return &result, nil
}

func (s *Storage) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	var key = newStorageKey(sessionUUID)

	if _, err := s.redis.ZRem(s.Context, key.requests(), requestUUID).Result(); err != nil {
		return false, err
	}

	return s.deleteKeys(key.request(requestUUID))
}

package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type Storage struct {
	ctx         context.Context
	rdb         *redis.Client
	ttl         time.Duration
	maxRequests uint16
	json        jsoniter.API
}

func NewStorage(ctx context.Context, rdb *redis.Client, sessionTTL time.Duration, maxRequests uint16) *Storage {
	return &Storage{
		ctx:         ctx,
		rdb:         rdb,
		ttl:         sessionTTL,
		maxRequests: maxRequests,
		json:        jsoniter.ConfigFastest,
	}
}

func (s *Storage) newUUID() string { return uuid.New().String() }

func (s *Storage) GetSession(sessionUUID string) (*storage.SessionData, error) {
	value, err := s.rdb.Get(s.ctx, storageKey(sessionUUID).session()).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	var sData = sessionData{}

	if jsonErr := s.json.Unmarshal(value, &sData); jsonErr != nil {
		return nil, jsonErr
	}

	return sData.toSharedStruct(sessionUUID), nil
}

func (s *Storage) CreateSession(wh *storage.WebHookResponse) (*storage.SessionData, error) {
	sData := sessionData{
		ResponseContent:     wh.Content,
		ResponseCode:        wh.Code,
		ResponseContentType: wh.ContentType,
		ResponseDelaySec:    wh.DelaySec,
		CreatedAtUnix:       time.Now().Unix(),
	}

	asJSON, jsonErr := s.json.Marshal(sData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	id := s.newUUID()

	if err := s.rdb.Set(s.ctx, storageKey(id).session(), asJSON, s.ttl).Err(); err != nil {
		return nil, err
	}

	return sData.toSharedStruct(id), nil
}

func (s *Storage) deleteKeys(keys ...string) (bool, error) {
	cmdResult := s.rdb.Del(s.ctx, keys...)

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
	return s.deleteKeys(storageKey(sessionUUID).session())
}

func (s *Storage) DeleteRequests(sessionUUID string) (bool, error) {
	key := storageKey(sessionUUID)

	// get request UUIDs, associated with session
	requestUUIDs, readErr := s.rdb.ZRangeByScore(s.ctx, key.requests(), &redis.ZRangeBy{
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
		now   = time.Now()
		rData = requestData{
			ClientAddr:    r.ClientAddr,
			Method:        r.Method,
			Content:       r.Content,
			Headers:       r.Headers,
			URI:           r.URI,
			CreatedAtUnix: now.Unix(),
		}
		key = storageKey(sessionUUID)
	)

	asJSON, jsonErr := s.json.Marshal(rData)
	if jsonErr != nil {
		return nil, jsonErr
	}

	id := s.newUUID()

	// save request data
	if _, err := s.rdb.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(s.ctx, key.requests(), &redis.Z{
			Score:  float64(now.UnixNano()),
			Member: id,
		})
		pipe.Set(s.ctx, key.request(id), asJSON, s.ttl)

		return nil
	}); err != nil {
		return nil, err
	}

	// read all stored request UUIDs
	requestUUIDs, readErr := s.rdb.ZRangeByScore(s.ctx, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if readErr != nil {
		return nil, readErr
	}

	// if currently we have more than allowed requests - remove unnecessary
	if len(requestUUIDs) > int(s.maxRequests) {
		if _, err := s.rdb.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
			for _, k := range requestUUIDs[:len(requestUUIDs)-int(s.maxRequests)] {
				pipe.ZRem(s.ctx, key.requests(), k)
				pipe.Del(s.ctx, key.request(k))
			}

			return nil
		}); err != nil {
			return nil, err
		}
	}

	// update expiring date
	if _, err := s.rdb.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		if len(requestUUIDs) > 0 {
			forUpdate := make([]string, 0, len(requestUUIDs))

			if len(requestUUIDs) > int(s.maxRequests) {
				forUpdate = requestUUIDs[len(requestUUIDs)-int(s.maxRequests):]
			} else {
				forUpdate = append(forUpdate, requestUUIDs...)
			}

			for _, k := range forUpdate {
				pipe.Expire(s.ctx, key.request(k), s.ttl)
			}
		}
		pipe.Expire(s.ctx, key.requests(), s.ttl)
		pipe.Expire(s.ctx, key.session(), s.ttl)

		return nil
	}); err != nil {
		return nil, err
	}

	return rData.toSharedStruct(id), nil
}

func (s *Storage) GetRequest(sessionUUID, requestUUID string) (*storage.RequestData, error) {
	value, err := s.rdb.Get(s.ctx, storageKey(sessionUUID).request(requestUUID)).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	rData := requestData{}
	if jsonErr := s.json.Unmarshal(value, &rData); jsonErr != nil {
		return nil, jsonErr
	}

	return rData.toSharedStruct(requestUUID), nil
}

func (s *Storage) GetAllRequests(sessionUUID string) (*[]storage.RequestData, error) {
	var key = storageKey(sessionUUID)

	if exists, existsErr := s.rdb.Exists(s.ctx, key.requests()).Result(); existsErr != nil {
		return nil, existsErr
	} else if exists == 0 {
		return nil, nil // not found
	}

	UUIDs, allErr := s.rdb.ZRangeByScore(s.ctx, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if allErr != nil {
		return nil, allErr
	}

	result := make([]storage.RequestData, 0, 8)

	if len(UUIDs) > 0 {
		// convert request UUIDs into storage keys
		keys := make([]string, len(UUIDs))
		for i, UUID := range UUIDs {
			keys[i] = key.request(UUID)
		}

		// read all requests in a one request
		rawRequests, gettingErr := s.rdb.MGet(s.ctx, keys...).Result()
		if gettingErr != nil {
			return nil, gettingErr
		}

		for i, UUID := range UUIDs {
			if json, ok := rawRequests[i].(string); ok {
				rData := requestData{}
				if err := s.json.Unmarshal([]byte(json), &rData); err == nil { // ignore errors with wrong json
					result = append(result, *rData.toSharedStruct(UUID))
				}
			}
		}
	}

	return &result, nil
}

func (s *Storage) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	var key = storageKey(sessionUUID)

	if _, err := s.rdb.ZRem(s.ctx, key.requests(), requestUUID).Result(); err != nil {
		return false, err
	}

	return s.deleteKeys(key.request(requestUUID))
}

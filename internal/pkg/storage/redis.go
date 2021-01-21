package storage

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

type redisSession struct {
	Uuid            string `json:"-"`
	RespContent     string `json:"resp_content"`
	RespCode        uint16 `json:"resp_code"`
	RespContentType string `json:"resp_content_type"`
	RespDelay       int64  `json:"resp_delay_nano"` // FIXME was `resp_delay_sec`, backwards incompatible
	TS              int64  `json:"created_at_unix"`
}

func (s *redisSession) UUID() string         { return s.Uuid }                     // UUID unique session ID.
func (s *redisSession) Content() string      { return s.RespContent }              // Content session server content.
func (s *redisSession) Code() uint16         { return s.RespCode }                 // Code default server response code.
func (s *redisSession) ContentType() string  { return s.RespContentType }          // ContentType response content type.
func (s *redisSession) Delay() time.Duration { return time.Duration(s.RespDelay) } // Delay before response sending.
func (s *redisSession) CreatedAt() time.Time { return time.Unix(s.TS, 0) }         // CreatedAt creation time.

type redisRequest struct {
	Uuid          string            `json:"-"`
	ReqClientAddr string            `json:"client_addr"`
	ReqMethod     string            `json:"method"`
	ReqContent    string            `json:"content"`
	ReqHeaders    map[string]string `json:"headers"`
	ReqURI        string            `json:"uri"`
	TS            int64             `json:"created_at_unix"`
}

func (r *redisRequest) UUID() string               { return r.Uuid }             // UUID returns unique request ID.
func (r *redisRequest) ClientAddr() string         { return r.ReqClientAddr }    // ClientAddr client hostname or IP.
func (r *redisRequest) Method() string             { return r.ReqMethod }        // Method HTTP method name.
func (r *redisRequest) Content() string            { return r.ReqContent }       // Content request body (payload).
func (r *redisRequest) Headers() map[string]string { return r.ReqHeaders }       // Headers HTTP request headers.
func (r *redisRequest) URI() string                { return r.ReqURI }           // URI Uniform Resource Identifier.
func (r *redisRequest) CreatedAt() time.Time       { return time.Unix(r.TS, 0) } // CreatedAt creation time.

type redisKey string

func (s redisKey) session() string          { return "webhook-tester:session:" + string(s) } // session data.
func (s redisKey) requests() string         { return s.session() + ":requests" }             // requests list.
func (s redisKey) request(id string) string { return s.session() + ":requests:" + id }       // request data.

// RedisStorage is redis storage implementation.
type RedisStorage struct {
	ctx         context.Context
	rdb         *redis.Client
	ttl         time.Duration
	maxRequests uint16
	json        jsoniter.API
}

// NewRedisStorage creates new redis storage instance.
func NewRedisStorage(ctx context.Context, rdb *redis.Client, sessionTTL time.Duration, maxRequests uint16) *RedisStorage { //nolint:lll
	return &RedisStorage{
		ctx:         ctx,
		rdb:         rdb,
		ttl:         sessionTTL,
		maxRequests: maxRequests,
		json:        jsoniter.ConfigFastest,
	}
}

func (s *RedisStorage) newUUID() string { return uuid.New().String() }

// GetSession returns session data.
func (s *RedisStorage) GetSession(uuid string) (Session, error) {
	value, err := s.rdb.Get(s.ctx, redisKey(uuid).session()).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	var sData = redisSession{}

	if jsonErr := s.json.Unmarshal(value, &sData); jsonErr != nil {
		return nil, jsonErr
	}

	sData.Uuid = uuid

	return &sData, nil
}

// CreateSession creates new session in storage using passed data.
func (s *RedisStorage) CreateSession(content string, code uint16, contentType string, delay time.Duration) (string, error) { //nolint:lll
	sData := redisSession{
		RespContent:     content,
		RespCode:        code,
		RespContentType: contentType,
		RespDelay:       delay.Nanoseconds(),
		TS:              time.Now().Unix(),
	}

	asJSON, jsonErr := s.json.Marshal(sData)
	if jsonErr != nil {
		return "", jsonErr
	}

	id := s.newUUID()

	if err := s.rdb.Set(s.ctx, redisKey(id).session(), asJSON, s.ttl).Err(); err != nil {
		return "", err
	}

	return id, nil
}

func (s *RedisStorage) deleteKeys(keys ...string) (bool, error) {
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

// DeleteSession deletes session with passed UUID.
func (s *RedisStorage) DeleteSession(uuid string) (bool, error) {
	return s.deleteKeys(redisKey(uuid).session())
}

// DeleteRequests deletes stored requests for session with passed UUID.
func (s *RedisStorage) DeleteRequests(sessionUUID string) (bool, error) {
	key := redisKey(sessionUUID)

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

	for i := 0; i < len(requestUUIDs); i++ {
		keys = append(keys, key.request(requestUUIDs[i]))
	}

	return s.deleteKeys(keys...)
}

// CreateRequest creates new request in storage using passed data.
func (s *RedisStorage) CreateRequest(sessionUUID, clientAddr, method, content, uri string, headers map[string]string) (string, error) { //nolint:funlen,lll
	var (
		now   = time.Now()
		rData = redisRequest{
			ReqClientAddr: clientAddr,
			ReqMethod:     method,
			ReqContent:    content,
			ReqHeaders:    headers,
			ReqURI:        uri,
			TS:            now.Unix(),
		}
		key = redisKey(sessionUUID)
	)

	asJSON, jsonErr := s.json.Marshal(rData)
	if jsonErr != nil {
		return "", jsonErr
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
		return "", err
	}

	// read all stored request UUIDs
	requestUUIDs, readErr := s.rdb.ZRangeByScore(s.ctx, key.requests(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if readErr != nil {
		return "", readErr
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
			return "", err
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

			for i := 0; i < len(forUpdate); i++ {
				pipe.Expire(s.ctx, key.request(forUpdate[i]), s.ttl)
			}
		}
		pipe.Expire(s.ctx, key.requests(), s.ttl)
		pipe.Expire(s.ctx, key.session(), s.ttl)

		return nil
	}); err != nil {
		return "", err
	}

	return id, nil
}

// GetRequest returns request data.
func (s *RedisStorage) GetRequest(sessionUUID, requestUUID string) (Request, error) {
	value, err := s.rdb.Get(s.ctx, redisKey(sessionUUID).request(requestUUID)).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // not found
		}

		return nil, err
	}

	rData := redisRequest{}
	if jsonErr := s.json.Unmarshal(value, &rData); jsonErr != nil {
		return nil, jsonErr
	}

	rData.Uuid = requestUUID

	return &rData, nil
}

// GetAllRequests returns all request as a slice of structures.
func (s *RedisStorage) GetAllRequests(sessionUUID string) ([]Request, error) {
	var key = redisKey(sessionUUID)

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

	result := make([]Request, 0, 8)

	if len(UUIDs) > 0 {
		// convert request UUIDs into storage keys
		keys := make([]string, len(UUIDs))

		for i := 0; i < len(UUIDs); i++ {
			keys[i] = key.request(UUIDs[i])
		}

		// read all requests in a one request
		rawRequests, gettingErr := s.rdb.MGet(s.ctx, keys...).Result()
		if gettingErr != nil {
			return nil, gettingErr
		}

		for i := 0; i < len(UUIDs); i++ {
			if json, ok := rawRequests[i].(string); ok {
				rData := redisRequest{}

				if err := s.json.Unmarshal([]byte(json), &rData); err == nil { // errors with wrong json ignored
					rData.Uuid = UUIDs[i]
					result = append(result, &rData)
				}
			}
		}
	}

	return result, nil
}

// DeleteRequest deletes stored request with passed session and request UUIDs.
func (s *RedisStorage) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	var key = redisKey(sessionUUID)

	if _, err := s.rdb.ZRem(s.ctx, key.requests(), requestUUID).Result(); err != nil {
		return false, err
	}

	return s.deleteKeys(key.request(requestUUID))
}

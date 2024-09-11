package storage

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

// Redis is redis storage implementation.
type Redis struct {
	ctx         context.Context
	rdb         *redis.Client
	ttl         time.Duration
	maxRequests uint16
}

// NewRedis creates new redis storage instance.
func NewRedis(ctx context.Context, rdb *redis.Client, sessionTTL time.Duration, maxRequests uint16) *Redis { //nolint:lll
	return &Redis{
		ctx:         ctx,
		rdb:         rdb,
		ttl:         sessionTTL,
		maxRequests: maxRequests,
	}
}

// GetSession returns session data.
func (s *Redis) GetSession(uuid string) (Session, error) {
	value, err := s.rdb.Get(s.ctx, redisKey(uuid).session()).Bytes()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // not found
		}

		return nil, err
	}

	var sData = redisSession{}

	if msgpackErr := msgpack.Unmarshal(value, &sData); msgpackErr != nil {
		return nil, msgpackErr
	}

	sData.Uuid = uuid

	return &sData, nil
}

// CreateSession creates new session in storage using passed data.
func (s *Redis) CreateSession(content []byte, code uint16, contentType string, delay time.Duration, sessionUUID ...string) (string, error) { //nolint:lll
	sData := redisSession{
		RespContent:     content,
		RespCode:        code,
		RespContentType: contentType,
		RespDelay:       delay.Nanoseconds(),
		TS:              time.Now().Unix(),
	}

	packed, msgpackErr := msgpack.Marshal(sData)
	if msgpackErr != nil {
		return "", msgpackErr
	}

	var id string

	if len(sessionUUID) == 1 && IsValidUUID(sessionUUID[0]) {
		id = sessionUUID[0]
	} else {
		id = NewUUID()
	}

	if err := s.rdb.Set(s.ctx, redisKey(id).session(), packed, s.ttl).Err(); err != nil {
		return "", err
	}

	return id, nil
}

func (s *Redis) deleteKeys(keys ...string) (bool, error) {
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
func (s *Redis) DeleteSession(uuid string) (bool, error) {
	return s.deleteKeys(redisKey(uuid).session())
}

// DeleteRequests deletes stored requests for session with passed UUID.
func (s *Redis) DeleteRequests(sessionUUID string) (bool, error) {
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

	for i := range requestUUIDs {
		keys = append(keys, key.request(requestUUIDs[i]))
	}

	return s.deleteKeys(keys...)
}

// CreateRequest creates new request in storage using passed data and updates expiration time for session and all
// stored requests for the session.
func (s *Redis) CreateRequest(sessionUUID, clientAddr, method, uri string, content []byte, headers map[string]string) (string, error) { //nolint:funlen,lll
	var (
		now = time.Now()
		key = redisKey(sessionUUID)
	)

	packed, msgpackErr := msgpack.Marshal(redisRequest{
		ReqClientAddr: clientAddr,
		ReqMethod:     method,
		ReqContent:    content,
		ReqHeaders:    headers,
		ReqURI:        uri,
		TS:            now.Unix(),
	})
	if msgpackErr != nil {
		return "", msgpackErr
	}

	id := NewUUID()

	// save request data
	if _, err := s.rdb.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(s.ctx, key.requests(), &redis.Z{
			Score:  float64(now.UnixNano()),
			Member: id,
		})
		pipe.Set(s.ctx, key.request(id), packed, s.ttl)

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

			for i := range forUpdate {
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
func (s *Redis) GetRequest(sessionUUID, requestUUID string) (Request, error) {
	value, err := s.rdb.Get(s.ctx, redisKey(sessionUUID).request(requestUUID)).Bytes()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // not found
		}

		return nil, err
	}

	rData := redisRequest{}
	if msgpackErr := msgpack.Unmarshal(value, &rData); msgpackErr != nil {
		return nil, msgpackErr
	}

	rData.Uuid = requestUUID

	return &rData, nil
}

// GetAllRequests returns all request as a slice of structures.
func (s *Redis) GetAllRequests(sessionUUID string) ([]Request, error) {
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

	result := make([]Request, 0, 8) //nolint:mnd

	if len(UUIDs) > 0 {
		// convert request UUIDs into storage keys
		keys := make([]string, len(UUIDs))

		for i := range UUIDs {
			keys[i] = key.request(UUIDs[i])
		}

		// read all requests in a one request
		rawRequests, gettingErr := s.rdb.MGet(s.ctx, keys...).Result()
		if gettingErr != nil {
			return nil, gettingErr
		}

		for i := range UUIDs {
			if packed, ok := rawRequests[i].(string); ok {
				rData := redisRequest{}

				if err := msgpack.Unmarshal([]byte(packed), &rData); err == nil { // errors with wrong data ignored
					rData.Uuid = UUIDs[i]
					result = append(result, &rData)
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].(*redisRequest).TS < result[j].(*redisRequest).TS
	})

	return result, nil
}

// DeleteRequest deletes stored request with passed session and request UUIDs.
func (s *Redis) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	var key = redisKey(sessionUUID)

	if _, err := s.rdb.ZRem(s.ctx, key.requests(), requestUUID).Result(); err != nil {
		return false, err
	}

	return s.deleteKeys(key.request(requestUUID))
}

type redisKey string

func (s redisKey) session() string          { return "webhook-tester:session:" + string(s) } // session data.
func (s redisKey) requests() string         { return s.session() + ":requests" }             // requests list.
func (s redisKey) request(id string) string { return s.session() + ":requests:" + id }       // request data.

type redisSession struct {
	Uuid            string `msgpack:"-"` //nolint:golint,stylecheck
	RespContent     []byte `msgpack:"c"`
	RespCode        uint16 `msgpack:"cd"`
	RespContentType string `msgpack:"ct"`
	RespDelay       int64  `msgpack:"d"`
	TS              int64  `msgpack:"t"`
}

func (s *redisSession) UUID() string         { return s.Uuid }                     // UUID unique session ID.
func (s *redisSession) Content() []byte      { return s.RespContent }              // Content session server content.
func (s *redisSession) Code() uint16         { return s.RespCode }                 // Code default server response code.
func (s *redisSession) ContentType() string  { return s.RespContentType }          // ContentType response content type.
func (s *redisSession) Delay() time.Duration { return time.Duration(s.RespDelay) } // Delay before response sending.
func (s *redisSession) CreatedAt() time.Time { return time.Unix(s.TS, 0) }         // CreatedAt creation time.

type redisRequest struct {
	Uuid          string            `msgpack:"-"` //nolint:golint,stylecheck
	ReqClientAddr string            `msgpack:"a"`
	ReqMethod     string            `msgpack:"m"`
	ReqContent    []byte            `msgpack:"c"`
	ReqHeaders    map[string]string `msgpack:"h"`
	ReqURI        string            `msgpack:"u"`
	TS            int64             `msgpack:"t"`
}

func (r *redisRequest) UUID() string               { return r.Uuid }             // UUID returns unique request ID.
func (r *redisRequest) ClientAddr() string         { return r.ReqClientAddr }    // ClientAddr client hostname or IP.
func (r *redisRequest) Method() string             { return r.ReqMethod }        // Method HTTP method name.
func (r *redisRequest) Content() []byte            { return r.ReqContent }       // Content request body (payload).
func (r *redisRequest) Headers() map[string]string { return r.ReqHeaders }       // Headers HTTP request headers.
func (r *redisRequest) URI() string                { return r.ReqURI }           // URI Uniform Resource Identifier.
func (r *redisRequest) CreatedAt() time.Time       { return time.Unix(r.TS, 0) } // CreatedAt creation time.

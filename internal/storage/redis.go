package storage

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 is used for non-cryptographic hashing of session/request IDs to Redis keys
	"encoding/hex"
	"encoding/json"
	"errors"
	"iter"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeysPrefix = "wht:v3:" // all Redis keys are prefixed with this to avoid collisions

// Redis is a Redis-backed sessions/requests storage implementation using the following layout.
// All keys for a session share the same Redis Cluster hash slot via the {{md5(sID)}} (`{` + text + `}`) hashing tag:
//
//	📂 {root}
//	├── wht:v3:{{md5(sID)}}                    // string key storing JSON-encoded [redisSession]
//	├── wht:v3:{{md5(sID)}}:requests:ordered   // (ZSET) sorted set of request IDs scored by creation time
//	├── wht:v3:{{md5(sID)}}:requests:data      // (HASH) key/field=md5(rID), value=JSON-encoded [redisRequest]
//	…
//
// IMPORTANT: The maximum Redis time accuracy is in milliseconds.
type Redis struct {
	client        redis.UniversalClient
	requestsLimit uint
	timeNow       func(context.Context) (time.Time, error)
}

var _ Storage = (*Redis)(nil) // compile-time interface assertion

// RedisOption allows to configure a [Redis] storage instance.
type RedisOption func(*Redis)

// WithRedisTimeNow overrides the clock used for all time-based operations.
func WithRedisTimeNow(fn func(context.Context) (time.Time, error)) RedisOption {
	return func(s *Redis) { s.timeNow = fn }
}

// NewRedis creates a new Redis storage instance.
// requestsLimit caps the number of captured requests stored per session (0 = unlimited).
func NewRedis(client redis.UniversalClient, requestsLimit uint, opts ...RedisOption) *Redis {
	s := &Redis{
		client:        client,
		requestsLimit: requestsLimit,
		timeNow:       func(ctx context.Context) (time.Time, error) { return client.Time(ctx).Result() },
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// NewSession implements [SessionStorage].
func (s *Redis) NewSession(
	ctx context.Context,
	sID string,
	response SessionResponse,
	ttl time.Duration,
) (*SessionMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now, tErr := s.timeNow(ctx)
	if tErr != nil {
		return nil, tErr
	}

	now = now.UTC()

	var expiresAt time.Time
	if ttl != NoExpiration {
		expiresAt = now.Add(ttl)
	}

	var headers []redisHeader
	if len(response.Headers) > 0 {
		headers = make([]redisHeader, len(response.Headers))
		for i, h := range response.Headers {
			headers[i] = redisHeader(h)
		}
	}

	data, encErr := json.Marshal(redisSession{
		Code:        response.Code,
		Headers:     headers,
		Body:        response.Body,
		DelayNs:     response.Delay.Nanoseconds(),
		CreatedAtUs: now.UnixMicro(),
	})
	if encErr != nil {
		return nil, encErr
	}

	// NX (Not eXists) - SET only if the key is absent; returns redis.Nil otherwise; combining NX with ExpireAt maps
	// to a single "SET key val NX EXAT <unix_sec>" command, so the key is created and its TTL is assigned atomically
	// in one round-trip
	args := redis.SetArgs{Mode: "NX"}
	if !expiresAt.IsZero() {
		args.ExpireAt = expiresAt
	}

	if err := s.client.SetArgs(ctx, s.keySession(sID), data, args).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionAlreadyExists
		}

		return nil, err
	}

	return &SessionMeta{CreatedAt: now, ExpiresAt: expiresAt}, nil
}

// GetSession implements [SessionStorage].
func (s *Redis) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sess, expiresAt, err := s.getSession(ctx, sID)
	if err != nil {
		return nil, err
	}

	return new(sess.toSession(expiresAt)), nil
}

// AddSessionTTL implements [SessionStorage].
func (s *Redis) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var expiresAtMs int64

	if howMuch != NoExpiration {
		now, err := s.timeNow(ctx)
		if err != nil {
			return err
		}

		expiresAtMs = now.Add(howMuch).UnixMilli()
	}

	var (
		sessKey        = s.keySession(sID)
		reqsOrderedKey = s.keyReqsOrdered(sID)
		reqsDataKey    = s.keyReqsData(sID)
	)

	// detect concurrent session deletion between the existence check and EXEC
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			if exists, err := tx.Exists(ctx, sessKey).Result(); err != nil {
				return err
			} else if exists == 0 {
				return ErrSessionNotFound
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				redisApplyExpiry(ctx, pipe, sessKey, expiresAtMs)
				redisApplyExpiry(ctx, pipe, reqsOrderedKey, expiresAtMs)
				redisApplyExpiry(ctx, pipe, reqsDataKey, expiresAtMs)

				return nil
			})

			return err
		}, sessKey); !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}
}

// DeleteSession implements [SessionStorage].
func (s *Redis) DeleteSession(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var (
		sessKey        = s.keySession(sID)
		reqsOrderedKey = s.keyReqsOrdered(sID)
		reqsDataKey    = s.keyReqsData(sID)
	)

	// detect concurrent deletion - return [ErrSessionNotFound] instead of silently no-oping
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			if exists, err := tx.Exists(ctx, sessKey).Result(); err != nil {
				return err
			} else if exists == 0 {
				return ErrSessionNotFound
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, sessKey, reqsOrderedKey, reqsDataKey)

				return nil
			})

			return err
		}, sessKey); !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}
}

// NewRequest implements [RequestStorage].
func (s *Redis) NewRequest( //nolint:gocognit,funlen
	ctx context.Context,
	sID, rID string,
	req CapturedRequest,
) (*RequestMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	headers := make([]redisHeader, len(req.Headers))
	for i, h := range req.Headers {
		headers[i] = redisHeader(h)
	}

	var now time.Time

	var (
		sessKey        = s.keySession(sID)
		reqField       = s.hash(rID)
		reqsOrderedKey = s.keyReqsOrdered(sID)
		reqsDataKey    = s.keyReqsData(sID)
	)

	// WATCH keyData (duplicate detection) + reqsKey (eviction correctness) - if either is modified between our reads
	// and EXEC, the transaction aborts and we retry. HSet and ZAdd are inside MULTI/EXEC so they either both land
	// or neither does - no partial state on failure
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			var tErr error

			now, tErr = s.timeNow(ctx)
			if tErr != nil {
				return tErr
			}

			now = now.UTC()

			// PEXPIRETIME sentinel values (go-redis stores them as raw nanosecond Durations, not scaled by precision):
			//   -2 = key does not exist
			//   -1 = key exists but has no associated expiry
			//   >0 = absolute Unix expiry in milliseconds (scaled by precision = time.Millisecond)
			sessExpire, ptErr := tx.PExpireTime(ctx, sessKey).Result()
			if ptErr != nil {
				return ptErr
			}

			if sessExpire == -2 {
				return ErrSessionNotFound
			}

			data, jErr := json.Marshal(redisRequest{
				ClientAddr:  req.ClientAddr,
				Method:      req.Method,
				Body:        req.Body,
				Headers:     headers,
				URL:         req.URL,
				CreatedAtUs: now.UnixMicro(),
			})
			if jErr != nil {
				return jErr
			}

			var reqExpireMs int64
			if sessExpire > 0 {
				reqExpireMs = sessExpire.Milliseconds()
			}

			if exists, err := tx.HExists(ctx, reqsDataKey, reqField).Result(); err != nil {
				return err
			} else if exists {
				return ErrRequestAlreadyExists
			}

			var toEvict []string

			if s.requestsLimit > 0 {
				count, cErr := tx.ZCard(ctx, reqsOrderedKey).Result()
				if cErr != nil {
					return cErr
				}

				limit := int64(s.requestsLimit) //nolint:gosec
				if count >= limit {
					excess := count - limit + 1 // +1 to make room for the new entry

					toEvict, cErr = tx.ZRange(ctx, reqsOrderedKey, 0, excess-1).Result()
					if cErr != nil {
						return cErr
					}
				}
			}

			_, txErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, reqsDataKey, reqField, data)
				pipe.ZAdd(ctx, reqsOrderedKey, redis.Z{Score: float64(now.UnixMicro()), Member: rID})
				redisApplyExpiry(ctx, pipe, reqsDataKey, reqExpireMs)
				redisApplyExpiry(ctx, pipe, reqsOrderedKey, reqExpireMs)

				for _, evictID := range toEvict {
					pipe.HDel(ctx, reqsDataKey, s.hash(evictID))
					pipe.ZRem(ctx, reqsOrderedKey, evictID)
				}

				return nil
			})

			return txErr
		}, reqsDataKey, reqsOrderedKey)

		if !errors.Is(err, redis.TxFailedErr) {
			if err != nil {
				return nil, err
			}

			return &RequestMeta{CreatedAt: now}, nil
		}
	}
}

// GetRequest implements [RequestStorage].
func (s *Redis) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var (
		existsCmd *redis.IntCmd
		hgetCmd   *redis.StringCmd
	)

	// pipeline EXISTS + HGet into one round-trip; redis.Nil from HGet is not a fatal pipeline error
	if _, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		existsCmd = pipe.Exists(ctx, s.keySession(sID))
		hgetCmd = pipe.HGet(ctx, s.keyReqsData(sID), s.hash(rID))

		return nil
	}); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	if existsCmd.Val() == 0 {
		return nil, ErrSessionNotFound
	}

	raw, err := hgetCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrRequestNotFound
		}

		return nil, err
	}

	var rdata redisRequest
	if err = json.Unmarshal(raw, &rdata); err != nil {
		return nil, err
	}

	return new(rdata.toRequest()), nil
}

// GetRequests implements [RequestStorage].
func (s *Redis) GetRequests( //nolint:gocognit
	ctx context.Context,
	sID string,
	errp *error,
) (iter.Seq2[string, Request], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if exists, err := s.client.Exists(ctx, s.keySession(sID)).Result(); err != nil {
		return nil, err
	} else if exists == 0 {
		return nil, ErrSessionNotFound
	}

	var (
		reqsOrderedKey = s.keyReqsOrdered(sID)
		reqsDataKey    = s.keyReqsData(sID)
	)

	const redisReqsBatchSize = 32 // how many requests to fetch in one batch

	return func(yield func(string, Request) bool) {
		var offset int64

		for {
			rIDs, gErr := s.client.ZRevRange(ctx, reqsOrderedKey, offset, offset+redisReqsBatchSize-1).Result()
			if gErr != nil {
				if errp != nil {
					*errp = gErr
				}

				return
			}

			if len(rIDs) == 0 {
				return
			}

			fields := make([]string, len(rIDs))
			for i, rID := range rIDs {
				fields[i] = s.hash(rID)
			}

			values, gErr := s.client.HMGet(ctx, reqsDataKey, fields...).Result()
			if gErr != nil {
				if errp != nil {
					*errp = gErr
				}

				return
			}

			for i, rID := range rIDs {
				if values[i] == nil {
					continue // evicted between ZRevRange and HMGet
				}

				raw, ok := values[i].(string)
				if !ok {
					continue
				}

				var rdata redisRequest
				if uErr := json.Unmarshal([]byte(raw), &rdata); uErr != nil {
					if errp != nil {
						*errp = uErr
					}

					return
				}

				if !yield(rID, rdata.toRequest()) {
					return
				}
			}

			if int64(len(rIDs)) < redisReqsBatchSize {
				return
			}

			offset += redisReqsBatchSize
		}
	}, nil
}

// DeleteRequest implements [RequestStorage].
func (s *Redis) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var (
		sessKey        = s.keySession(sID)
		reqField       = s.hash(rID)
		reqsOrderedKey = s.keyReqsOrdered(sID)
		reqsDataKey    = s.keyReqsData(sID)
	)

	// detect concurrent session deletion between the existence check and EXEC
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var hdelCmd *redis.IntCmd

		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			if exists, err := tx.Exists(ctx, sessKey).Result(); err != nil {
				return err
			} else if exists == 0 {
				return ErrSessionNotFound
			}

			// HDel and ZRem are inside MULTI/EXEC so neither is visible without the other
			_, txErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				hdelCmd = pipe.HDel(ctx, reqsDataKey, reqField)
				pipe.ZRem(ctx, reqsOrderedKey, rID)

				return nil
			})

			return txErr
		}, sessKey)

		if !errors.Is(err, redis.TxFailedErr) {
			if err != nil {
				return err
			}

			if hdelCmd.Val() == 0 {
				return ErrRequestNotFound
			}

			return nil
		}
	}
}

// DeleteAllRequests implements [RequestStorage].
func (s *Redis) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var (
		sessKey        = s.keySession(sID)
		reqsOrderedKey = s.keyReqsOrdered(sID)
	)

	// WATCH sessKey and reqsKey - sessKey detects concurrent session deletion (for correct error propagation);
	// reqsKey detects concurrent NewRequest additions, so a "write in flight" isn't silently lost on our retry
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			if exists, err := tx.Exists(ctx, sessKey).Result(); err != nil {
				return err
			} else if exists == 0 {
				return ErrSessionNotFound
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, reqsOrderedKey, s.keyReqsData(sID))

				return nil
			})

			return err
		}, sessKey, reqsOrderedKey); !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}
}

// getSession fetches and decodes the session JSON. Returns (nil, nil, ErrSessionNotFound) for a missing key.
// On success (err == nil) the [time.Time] pointer is always non-nil - a zero value means no expiry is set,
// a non-zero value carries the authoritative Redis key expiration time.
func (s *Redis) getSession(ctx context.Context, sID string) (*redisSession, *time.Time, error) {
	var (
		getCmd     *redis.StringCmd
		pexpireCmd *redis.DurationCmd
	)

	sessKey := s.keySession(sID)

	if _, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		getCmd = pipe.Get(ctx, sessKey)
		pexpireCmd = pipe.PExpireTime(ctx, sessKey)

		return nil
	}); err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	raw, err := getCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil, ErrSessionNotFound
		}

		return nil, nil, err
	}

	var sess redisSession
	if err = json.Unmarshal(raw, &sess); err != nil {
		return nil, nil, err
	}

	// PEXPIRETIME returns -1 if no expiry, -2 if key doesn't exist (already handled by GET above)
	var expiresAt time.Time // accuracy is in milliseconds

	if d := pexpireCmd.Val(); d > 0 {
		expiresAt = time.UnixMilli(d.Milliseconds()).UTC()
	}

	return &sess, &expiresAt, nil
}

// redisApplyExpiry sets PEXPIREAT or PERSIST on key using a pipeline.
func redisApplyExpiry(ctx context.Context, pipe redis.Pipeliner, key string, expiresAtMs int64) {
	if expiresAtMs > 0 {
		// set a timeout on key; after the timeout has expired, the key will automatically be deleted (time is in
		// milliseconds)
		pipe.PExpireAt(ctx, key, time.UnixMilli(expiresAtMs))
	} else {
		// remove the existing timeout on key, turning the key from volatile (a key with an expiry set) to persistent
		// (a key that will never expire as no timeout is associated)
		pipe.Persist(ctx, key)
	}
}

// hash returns the lowercase hex-encoded MD5 of the input string.
// Used for generating Redis keys from session/request IDs.
func (*Redis) hash(v string) string { h := md5.Sum([]byte(v)); return hex.EncodeToString(h[:]) } //nolint:nlreturn,gosec

// keySession returns the Redis key for session data.
// The {hash} portion is a Redis "hash" tag that pins all session keys to the same cluster slot.
//
// Example: "wht:v3:{5d41402abc4b2a76b9719d911017c592}" for sID "hello".
func (s *Redis) keySession(sID string) string { return redisKeysPrefix + "{" + s.hash(sID) + "}" }

// keyReqsOrdered returns the Redis key for the sorted set of request IDs, scored by creation time (newest-last).
//
// Example: "wht:v3:{5d41402abc4b2a76b9719d911017c592}:requests:ordered" for sID "hello".
func (s *Redis) keyReqsOrdered(sID string) string {
	return s.keySession(sID) + ":requests:ordered"
}

// keyReqsData returns the Redis key for the hash storing request payloads.
// Each field is md5(rID) and each value is a JSON-encoded [redisRequest].
//
// Example: "wht:v3:{5d41402abc4b2a76b9719d911017c592}:requests:data" for sID "hello".
func (s *Redis) keyReqsData(sID string) string {
	return s.keySession(sID) + ":requests:data"
}

// --------------------------------------------------------------------------------------------------------------------

type redisHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type redisSession struct {
	Code        uint16        `json:"code"`
	Headers     []redisHeader `json:"headers,omitempty"`
	Body        []byte        `json:"body,omitempty"`
	DelayNs     int64         `json:"delay_ns,omitempty"`
	CreatedAtUs int64         `json:"created_at_us"`
}

func (d *redisSession) toSession(expiresAt *time.Time) Session {
	var headers []ResponseHeader
	if len(d.Headers) > 0 {
		headers = make([]ResponseHeader, len(d.Headers))
		for i, h := range d.Headers {
			headers[i] = ResponseHeader(h)
		}
	}

	return Session{
		Response: SessionResponse{
			Code:    d.Code,
			Headers: headers,
			Body:    d.Body,
			Delay:   time.Duration(d.DelayNs),
		},
		Meta: SessionMeta{
			CreatedAt: time.UnixMicro(d.CreatedAtUs).UTC(),
			ExpiresAt: *expiresAt,
		},
	}
}

type redisRequest struct {
	ClientAddr  string        `json:"client_addr,omitempty"`
	Method      string        `json:"method,omitempty"`
	Body        []byte        `json:"body,omitempty"`
	Headers     []redisHeader `json:"headers,omitempty"`
	URL         string        `json:"url,omitempty"`
	CreatedAtUs int64         `json:"created_at_us"`
}

func (d *redisRequest) toRequest() Request {
	var headers []RequestHeader
	if len(d.Headers) > 0 {
		headers = make([]RequestHeader, len(d.Headers))
		for i, h := range d.Headers {
			headers[i] = RequestHeader(h)
		}
	}

	return Request{
		Data: CapturedRequest{
			ClientAddr: d.ClientAddr,
			Method:     d.Method,
			Body:       d.Body,
			Headers:    headers,
			URL:        d.URL,
		},
		Meta: RequestMeta{CreatedAt: time.UnixMicro(d.CreatedAtUs).UTC()},
	}
}

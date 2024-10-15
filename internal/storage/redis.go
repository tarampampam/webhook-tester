package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type (
	Redis struct {
		sessionTTL  time.Duration
		maxRequests uint32
		client      redis.Cmdable
		ctx         context.Context
	}
)

var ( // ensure interface implementation
	_ Storage = (*Redis)(nil)
)

type RedisOption func(*Redis)

func NewRedis(ctx context.Context, c redis.Cmdable, sTTL time.Duration, maxReq uint32, opts ...RedisOption) *Redis {
	var s = Redis{
		sessionTTL:  sTTL,
		maxRequests: maxReq,
		client:      c,
		ctx:         ctx,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

// sessionKey returns the key for the session data.
func (s *Redis) sessionKey(sID string) string { return "webhook-tester-v2:session:" + sID }

// requestsKey returns the key for the requests list.
func (s *Redis) requestsKey(sID string) string { return s.sessionKey(sID) + ":requests" }

// requestKey returns the key for the request data.
func (s *Redis) requestKey(sID, rID string) string { return s.sessionKey(sID) + ":requests:" + rID }

// newID generates a new (unique) ID.
func (s *Redis) newID() string { return uuid.New().String() }

func (s *Redis) unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (s *Redis) marshal(v any) ([]byte, error)      { return json.Marshal(v) }

func (s *Redis) NewSession(session Session) (sID string, _ error) {
	sID, session.CreatedAt.Time = s.newID(), time.Now()

	data, mErr := s.marshal(session)
	if mErr != nil {
		return "", mErr
	}

	if err := s.client.Set(s.ctx, s.sessionKey(sID), data, s.sessionTTL).Err(); err != nil {
		return "", err
	}

	return sID, nil
}

func (s *Redis) GetSession(sID string) (*Session, error) {
	data, rErr := s.client.Get(s.ctx, s.sessionKey(sID)).Bytes()
	if rErr != nil {
		if errors.Is(rErr, redis.Nil) {
			return nil, ErrSessionNotFound
		}

		return nil, rErr
	}

	var session Session
	if uErr := s.unmarshal(data, &session); uErr != nil {
		return nil, uErr
	}

	return &session, nil
}

func (s *Redis) DeleteSession(sID string) error {
	if result := s.client.Del(s.ctx, s.sessionKey(sID)); result.Err() != nil {
		return result.Err()
	} else if count, rErr := result.Result(); rErr != nil {
		return rErr
	} else if count == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (s *Redis) NewRequest(sID string, r Request) (rID string, _ error) { //nolint:funlen
	// check the session existence
	if _, err := s.GetSession(sID); err != nil {
		return "", err
	}

	rID, r.CreatedAt.Time = s.newID(), time.Now()

	data, mErr := s.marshal(r)
	if mErr != nil {
		return "", mErr
	}

	// save the request data
	if _, err := s.client.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(s.ctx, s.requestsKey(sID), &redis.Z{Score: float64(r.CreatedAt.UnixNano()), Member: rID})
		pipe.Set(s.ctx, s.requestKey(sID, rID), data, s.sessionTTL)

		return nil
	}); err != nil {
		return "", err
	}

	// read all stored request UUIDs
	ids, rErr := s.client.ZRangeByScore(s.ctx, s.requestsKey(sID), &redis.ZRangeBy{Min: "-inf", Max: "+inf"}).Result()
	if rErr != nil {
		return "", rErr
	}

	// if we have too many requests - remove unnecessary
	if len(ids) > int(s.maxRequests) {
		if _, err := s.client.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
			for _, id := range ids[:len(ids)-int(s.maxRequests)] {
				pipe.ZRem(s.ctx, s.requestsKey(sID), id)
				pipe.Del(s.ctx, s.requestKey(sID, id))
			}

			return nil
		}); err != nil {
			return "", err
		}
	}

	// update the expiration date
	if _, err := s.client.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		if len(ids) > 0 {
			var forUpdate = make([]string, 0, len(ids))

			if len(ids) > int(s.maxRequests) {
				forUpdate = ids[len(ids)-int(s.maxRequests):]
			} else {
				forUpdate = append(forUpdate, ids...)
			}

			for i := range forUpdate {
				pipe.Expire(s.ctx, s.requestKey(sID, forUpdate[i]), s.sessionTTL)
			}
		}

		pipe.Expire(s.ctx, s.requestsKey(sID), s.sessionTTL)
		pipe.Expire(s.ctx, s.sessionKey(sID), s.sessionTTL)

		return nil
	}); err != nil {
		return "", err
	}

	return rID, nil
}

func (s *Redis) GetRequest(sID, rID string) (*Request, error) {
	// check the session existence
	if _, err := s.GetSession(sID); err != nil {
		return nil, err
	}

	data, rErr := s.client.Get(s.ctx, s.requestKey(sID, rID)).Bytes()
	if rErr != nil {
		if errors.Is(rErr, redis.Nil) {
			return nil, ErrRequestNotFound
		}

		return nil, rErr
	}

	var request Request
	if uErr := s.unmarshal(data, &request); uErr != nil {
		return nil, uErr
	}

	return &request, nil
}

func (s *Redis) GetAllRequests(sID string) (map[string]Request, error) {
	// check the session existence
	if _, err := s.GetSession(sID); err != nil {
		return nil, err
	}

	// read all stored request IDs
	ids, rErr := s.client.ZRangeByScore(s.ctx, s.requestsKey(sID), &redis.ZRangeBy{Min: "-inf", Max: "+inf"}).Result()
	if rErr != nil {
		return nil, rErr
	}

	if len(ids) == 0 {
		return make(map[string]Request), nil
	}

	var (
		all  = make(map[string]Request, len(ids))
		keys = make([]string, len(ids))
	)

	// convert request IDs to keys
	for i, id := range ids {
		keys[i] = s.requestKey(sID, id)
	}

	// read all request data
	data, mErr := s.client.MGet(s.ctx, keys...).Result()
	if mErr != nil {
		return nil, mErr
	}

	for i, d := range data {
		if d == nil {
			continue
		}

		var request Request
		if uErr := s.unmarshal([]byte(d.(string)), &request); uErr != nil {
			return nil, uErr
		}

		all[ids[i]] = request
	}

	return all, nil
}

func (s *Redis) DeleteRequest(sID, rID string) error {
	// check the session existence
	if _, err := s.GetSession(sID); err != nil {
		return err
	}

	var deleted *redis.IntCmd

	// delete the request
	if _, err := s.client.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZRem(s.ctx, s.requestsKey(sID), rID)
		deleted = pipe.Del(s.ctx, s.requestKey(sID, rID))

		return nil
	}); err != nil {
		return err
	}

	if deleted.Val() == 0 {
		return ErrRequestNotFound
	}

	return nil
}

func (s *Redis) DeleteAllRequests(sID string) error {
	// check the session existence
	if _, err := s.GetSession(sID); err != nil {
		return err
	}

	// read all stored request IDs
	ids, rErr := s.client.ZRangeByScore(s.ctx, s.requestsKey(sID), &redis.ZRangeBy{Min: "-inf", Max: "+inf"}).Result()
	if rErr != nil {
		return rErr
	}

	// delete all requests
	if _, err := s.client.Pipelined(s.ctx, func(pipe redis.Pipeliner) error {
		for _, id := range ids {
			pipe.Del(s.ctx, s.requestKey(sID, id))
		}

		pipe.Del(s.ctx, s.requestsKey(sID))

		return nil
	}); err != nil {
		return err
	}

	return nil
}

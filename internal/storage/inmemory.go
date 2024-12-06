package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type (
	InMemory struct {
		sessionTTL      time.Duration
		maxRequests     uint32
		sessions        syncMap[ /* sID */ string, *sessionData]
		cleanupInterval time.Duration

		// this function returns the current time, it's used to mock the time in tests
		timeNow TimeFunc

		close  chan struct{}
		closed atomic.Bool
	}

	sessionData struct {
		sync.Mutex
		session  Session
		requests syncMap[ /* rID */ string, Request]
	}
)

var ( // ensure interface implementation
	_ Storage   = (*InMemory)(nil)
	_ io.Closer = (*InMemory)(nil)
)

type InMemoryOption func(*InMemory)

// WithInMemoryCleanupInterval sets the cleanup interval for expired sessions.
func WithInMemoryCleanupInterval(v time.Duration) InMemoryOption {
	return func(s *InMemory) { s.cleanupInterval = v }
}

// WithInMemoryTimeNow sets the function that returns the current time.
func WithInMemoryTimeNow(fn TimeFunc) InMemoryOption { return func(s *InMemory) { s.timeNow = fn } }

// NewInMemory creates a new in-memory storage with the given session TTL and the maximum number of stored requests.
// Note that the cleanup goroutine is started automatically if the cleanup interval is greater than zero.
// To stop the cleanup goroutine and close the storage, call the InMemory.Close method.
func NewInMemory(sessionTTL time.Duration, maxRequests uint32, opts ...InMemoryOption) *InMemory {
	var s = InMemory{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		close:           make(chan struct{}),
		cleanupInterval: time.Second, // default cleanup interval
		timeNow:         defaultTimeFunc,
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.cleanupInterval > time.Duration(0) {
		go s.cleanup(context.Background()) // start cleanup goroutine
	}

	return &s
}

// newID generates a new (unique) ID.
func (*InMemory) newID() string { return uuid.New().String() }

func (s *InMemory) cleanup(ctx context.Context) {
	var timer = time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	defer func() { // cleanup on exit
		s.sessions.Range(func(sID string, _ *sessionData) bool {
			_ = s.DeleteSession(ctx, sID)

			return true
		})
	}()

	for {
		select {
		case <-s.close: // close signal received
			return
		case <-timer.C:
			var now = s.timeNow()

			s.sessions.Range(func(sID string, data *sessionData) bool {
				data.Lock()
				var expiresAt = data.session.ExpiresAt
				data.Unlock()

				if expiresAt.Before(now) {
					_ = s.DeleteSession(ctx, sID)
				}

				return true
			})

			timer.Reset(s.cleanupInterval)
		}
	}
}

// isSessionExists checks if the session with the specified ID exists and is not expired.
func (s *InMemory) isSessionExists(sID string) bool {
	data, ok := s.sessions.Load(sID)
	if !ok {
		return false
	}

	data.Lock()
	var expiresAt = data.session.ExpiresAt
	data.Unlock()

	// TODO: remove expired sessions automatically?

	return expiresAt.After(s.timeNow())
}

// isOpenAndNotDone checks if the storage is open and the context is not done.
func (s *InMemory) isOpenAndNotDone(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	return nil
}

func (s *InMemory) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err // context is done
	}

	var now = s.timeNow()

	if len(id) > 0 { //nolint:nestif // use the specified ID
		if len(id[0]) == 0 {
			return "", errors.New("empty session ID")
		}

		sID = id[0]

		// check if the session with the specified ID already exists
		if data, ok := s.sessions.Load(sID); ok {
			return "", fmt.Errorf("session %s already exists", sID)
		} else {
			// check if the session with the specified ID has expired
			data.Lock()
			expiresAt := data.session.ExpiresAt
			data.Unlock()

			if expiresAt.After(now) {
				if dErr := s.DeleteSession(ctx, sID); dErr != nil {
					return "", dErr
				}
			}
		}
	} else {
		sID = s.newID() // generate a new ID
	}

	session.CreatedAtUnixMilli, session.ExpiresAt = now.UnixMilli(), now.Add(s.sessionTTL)

	s.sessions.Store(sID, &sessionData{session: session})

	return
}

func (s *InMemory) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	data, ok := s.sessions.Load(sID)
	if !ok {
		return nil, ErrSessionNotFound // not found
	}

	data.Lock()
	var expiresAt = data.session.ExpiresAt
	data.Unlock()

	if expiresAt.Before(s.timeNow()) {
		s.sessions.Delete(sID)

		return nil, ErrSessionNotFound // session has been expired
	}

	return &data.session, nil
}

func (s *InMemory) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if !s.isSessionExists(sID) {
		return ErrSessionNotFound // session not found
	}

	data, ok := s.sessions.Load(sID)
	if !ok {
		return ErrSessionNotFound // like a fuse, because we already checked it
	}

	data.Lock()
	data.session.ExpiresAt = data.session.ExpiresAt.Add(howMuch)
	data.Unlock()

	return nil
}

func (s *InMemory) DeleteSession(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if data, ok := s.sessions.LoadAndDelete(sID); !ok {
		return ErrSessionNotFound // session not found
	} else {
		data.requests.Range(func(rID string, _ Request) bool { // delete all session requests
			data.requests.Delete(rID)

			return true
		})
	}

	return nil
}

func (s *InMemory) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err
	}

	if !s.isSessionExists(sID) {
		return "", ErrSessionNotFound // session not found
	}

	data, ok := s.sessions.Load(sID)
	if !ok {
		return "", ErrSessionNotFound // like a fuse, because we already checked it
	}

	rID, r.CreatedAtUnixMilli = s.newID(), s.timeNow().UnixMilli()

	data.requests.Store(rID, r)

	{ // limit stored requests count
		type rq struct { // a runtime representation of the request, used for sorting
			id string
			ts int64
		}

		var all = make([]rq, 0) // a slice for all session requests

		data.requests.Range(func(id string, req Request) bool { // iterate over all session requests and fill the slice
			all = append(all, rq{id, req.CreatedAtUnixMilli})

			return true
		})

		if len(all) > int(s.maxRequests) { // if the number of requests exceeds the limit
			sort.Slice(all, func(i, j int) bool { return all[i].ts > all[j].ts }) // sort requests by creation time

			for i := int(s.maxRequests); i < len(all); i++ { // delete the oldest requests
				data.requests.Delete(all[i].id)
			}
		}
	}

	return
}

func (s *InMemory) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	if !s.isSessionExists(sID) {
		return nil, ErrSessionNotFound // session not found
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return nil, ErrSessionNotFound // like a fuse, because we already checked it
	}

	if request, ok := session.requests.Load(rID); ok {
		return &request, nil
	}

	return nil, ErrRequestNotFound // request not found
}

func (s *InMemory) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	if !s.isSessionExists(sID) {
		return nil, ErrSessionNotFound // session not found
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return nil, ErrSessionNotFound // like a fuse, because we already checked it
	}

	var all = make(map[string]Request)

	session.requests.Range(func(id string, req Request) bool {
		all[id] = req

		return true
	})

	return all, nil
}

func (s *InMemory) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if !s.isSessionExists(sID) {
		return ErrSessionNotFound // session not found
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return ErrSessionNotFound // like a fuse, because we already checked it
	}

	if _, ok := session.requests.LoadAndDelete(rID); ok {
		return nil
	}

	return ErrRequestNotFound // request not found
}

func (s *InMemory) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if !s.isSessionExists(sID) {
		return ErrSessionNotFound // session not found
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return ErrSessionNotFound // like a fuse, because we already checked it
	}

	// delete all session requests
	session.requests.Range(func(rID string, _ Request) bool {
		session.requests.Delete(rID)

		return true
	})

	return nil
}

// Close closes the storage and stops the cleanup goroutine. Any further calls to the storage methods will
// return ErrClosed.
func (s *InMemory) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		close(s.close)

		return nil
	}

	return ErrClosed
}

// syncMap is a thread-safe map with strong-typed keys and values.
type syncMap[K comparable, V any] struct{ m sync.Map }

// Delete deletes the value for a key.
func (m *syncMap[K, V]) Delete(key K) { m.m.Delete(key) }

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *syncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}

	return v.(V), ok
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *syncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}

	return v.(V), loaded
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *syncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool { return f(key.(K), value.(V)) })
}

// Store sets the value for a key.
func (m *syncMap[K, V]) Store(key K, value V) { m.m.Store(key, value) }

package storage

import (
	"context"
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

		close  chan struct{}
		closed atomic.Bool
	}

	sessionData struct {
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

// NewInMemory creates a new in-memory storage with the given session TTL and the maximum number of stored requests.
// Note that the cleanup goroutine is started automatically if the cleanup interval is greater than zero.
// To stop the cleanup goroutine and close the storage, call the InMemory.Close method.
func NewInMemory(sessionTTL time.Duration, maxRequests uint32, opts ...InMemoryOption) *InMemory {
	var s = InMemory{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		close:           make(chan struct{}),
		cleanupInterval: time.Second, // default cleanup interval
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.cleanupInterval > time.Duration(0) {
		go s.cleanup() // start cleanup goroutine
	}

	return &s
}

// newID generates a new (unique) ID.
func (s *InMemory) newID() string { return uuid.New().String() }

func (s *InMemory) cleanup() {
	var timer = time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	var ctx = context.Background()

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
			var now = time.Now()

			s.sessions.Range(func(sID string, data *sessionData) bool {
				if data.session.CreatedAt.Add(s.sessionTTL).Before(now) {
					_ = s.DeleteSession(ctx, sID)
				}

				return true
			})

			timer.Reset(s.cleanupInterval)
		}
	}
}

func (s *InMemory) NewSession(ctx context.Context, session Session) (sID string, _ error) {
	if err := ctx.Err(); err != nil {
		return "", err // context is done
	} else if s.closed.Load() {
		return "", ErrClosed // storage is closed
	}

	sID, session.CreatedAt.Time = s.newID(), time.Now()

	s.sessions.Store(sID, &sessionData{session: session})

	return
}

func (s *InMemory) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	data, ok := s.sessions.Load(sID)
	if !ok {
		return nil, ErrSessionNotFound // not found
	}

	if data.session.CreatedAt.Add(s.sessionTTL).Before(time.Now()) {
		s.sessions.Delete(sID)

		return nil, ErrSessionNotFound // session has been expired
	}

	return &data.session, nil
}

func (s *InMemory) DeleteSession(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
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
	if err := ctx.Err(); err != nil {
		return "", err // context is done
	} else if s.closed.Load() {
		return "", ErrClosed // storage is closed
	}

	data, ok := s.sessions.Load(sID)
	if !ok {
		return "", ErrSessionNotFound // session not found
	}

	rID, r.CreatedAt.Time = s.newID(), time.Now()

	data.requests.Store(rID, r)

	{ // limit stored requests count
		type rq struct { // a runtime representation of the request, used for sorting
			id string
			ts int64
		}

		var all = make([]rq, 0) // a slice for all session requests

		data.requests.Range(func(id string, req Request) bool { // iterate over all session requests and fill the slice
			all = append(all, rq{id, req.CreatedAt.UnixNano()})

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
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return nil, ErrSessionNotFound // session not found
	}

	if request, ok := session.requests.Load(rID); ok {
		return &request, nil
	}

	return nil, ErrRequestNotFound // request not found
}

func (s *InMemory) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return nil, ErrSessionNotFound // session not found
	}

	var all = make(map[string]Request)

	session.requests.Range(func(id string, req Request) bool {
		all[id] = req

		return true
	})

	return all, nil
}

func (s *InMemory) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return ErrSessionNotFound // session not found
	}

	if _, ok := session.requests.LoadAndDelete(rID); ok {
		return nil
	}

	return ErrRequestNotFound // request not found
}

func (s *InMemory) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	session, sessionOk := s.sessions.Load(sID)
	if !sessionOk {
		return ErrSessionNotFound // session not found
	}

	// delete all session requests
	session.requests.Range(func(rID string, _ Request) bool {
		session.requests.Delete(rID)

		return true
	})

	return nil // no requests
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

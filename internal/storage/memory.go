package storage

import (
	"context"
	"io"
	"iter"
	"slices"
	"sync"
	"time"
)

// Memory is an in-memory sessions/requests storage implementation.
//
// All data is lost when the process exits. A background goroutine evicts expired sessions on a configurable interval;
// it stops when Close is called or ctx is canceled.
// After either event all methods return [ErrClosed].
type Memory struct {
	sessions      memSessionMap
	requestsLimit uint // how many captured requests to store per session before evicting old ones (0 = unlimited)
	timeNow       func() time.Time
	cleanupTick   time.Duration

	closeOnce     sync.Once
	closeSignalCh chan struct{}
	closedCh      chan struct{}
}

var ( // compile-time interface assertions
	_ Storage   = (*Memory)(nil)
	_ io.Closer = (*Memory)(nil)
)

// MemoryOption allows to configure a Memory storage instance.
type MemoryOption func(*Memory)

// WithMemoryTimeNow sets the function that returns the current time.
func WithMemoryTimeNow(fn func() time.Time) MemoryOption { return func(s *Memory) { s.timeNow = fn } }

// WithMemoryCleanupInterval sets the interval between expired-session cleanup sweeps.
// Defaults to 1 second.
func WithMemoryCleanupInterval(d time.Duration) MemoryOption {
	return func(s *Memory) { s.cleanupTick = d }
}

// NewMemory creates a new in-memory storage instance.
// `requestsLimit` is the maximum number of captured requests stored per session (0 = unlimited).
// The cleanup goroutine stops when Close is called or ctx is canceled; either event closes the storage.
func NewMemory(ctx context.Context, requestsLimit uint, opts ...MemoryOption) *Memory {
	s := &Memory{
		sessions:      memSessionMap{data: make(map[string]*memSession)},
		requestsLimit: requestsLimit,
		timeNow:       time.Now,
		cleanupTick:   time.Second,
		closeSignalCh: make(chan struct{}),
		closedCh:      make(chan struct{}),
	}

	for _, o := range opts {
		o(s)
	}

	// run cleanup loop in background; it will exit when the storage is closed or the context is canceled
	go func() {
		defer func() {
			s.closeOnce.Do(func() { close(s.closeSignalCh) }) // ensure ErrClosed is returned after ctx cancel
			close(s.closedCh)
		}()

		ticker := time.NewTicker(s.cleanupTick)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.closeSignalCh:
				return
			case <-ticker.C:
				s.sessions.mu.Lock()
				now := s.timeNow()

				for id, sess := range s.sessions.data {
					sess.mu.Lock()
					if sess.isExpired(now) {
						clear(sess.requests) // make GC a bit happier
						delete(s.sessions.data, id)
					}
					sess.mu.Unlock()
				}

				s.sessions.mu.Unlock()
			}
		}
	}()

	return s
}

// checkOpen returns [ErrClosed] if the storage has been closed, or ctx.Err() if the context is done.
func (s *Memory) checkOpen(ctx context.Context) error {
	select {
	case <-s.closeSignalCh:
		return ErrClosed
	default:
		return ctx.Err()
	}
}

// NewSession implements [SessionStorage].
func (s *Memory) NewSession(
	ctx context.Context,
	sID string,
	response SessionResponse,
	ttl time.Duration,
) (*SessionMeta, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.sessions.mu.Lock()
	defer s.sessions.mu.Unlock()

	now := s.timeNow()

	if existing, ok := s.sessions.data[sID]; ok {
		existing.mu.RLock()
		expired := existing.isExpired(now)
		existing.mu.RUnlock()

		if !expired {
			return nil, ErrSessionAlreadyExists
		}
	}

	var expiresAt time.Time
	if ttl != NoExpiration {
		expiresAt = now.Add(ttl)
	}

	s.sessions.data[sID] = &memSession{
		response:  cloneSessionResponse(response),
		createdAt: now,
		expiresAt: expiresAt,
		requests:  make(map[string]*memRequest, s.requestsLimit),
	}

	return &SessionMeta{CreatedAt: now, ExpiresAt: expiresAt}, nil
}

// GetSession implements [SessionStorage].
func (s *Memory) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	sess.mu.RLock()
	defer sess.mu.RUnlock()

	if sess.isExpired(s.timeNow()) {
		return nil, ErrSessionNotFound
	}

	return new(sess.toSession()), nil
}

// AddSessionTTL implements [SessionStorage].
func (s *Memory) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	if howMuch == NoExpiration {
		sess.expiresAt = time.Time{}
	} else {
		sess.expiresAt = s.timeNow().Add(howMuch)
	}

	return nil
}

// DeleteSession implements [SessionStorage].
func (s *Memory) DeleteSession(ctx context.Context, sID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.sessions.mu.Lock()
	defer s.sessions.mu.Unlock()

	sess, ok := s.sessions.data[sID]
	if !ok {
		return ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	clear(sess.requests) // make GC a bit happier
	delete(s.sessions.data, sID)

	return nil
}

// NewRequest implements [RequestStorage].
func (s *Memory) NewRequest(ctx context.Context, sID, rID string, req CapturedRequest) (*RequestMeta, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	now := s.timeNow()

	if sess.isExpired(now) {
		return nil, ErrSessionNotFound
	}

	if _, exists := sess.requests[rID]; exists {
		return nil, ErrRequestAlreadyExists
	}

	if s.requestsLimit > 0 && uint(len(sess.requests)) >= s.requestsLimit {
		var (
			oldestID string
			oldestAt time.Time
		)

		for id, r := range sess.requests {
			if oldestID == "" || r.createdAt.Before(oldestAt) {
				oldestID, oldestAt = id, r.createdAt
			}
		}

		if oldestID != "" {
			delete(sess.requests, oldestID)
		}
	}

	sess.requests[rID] = &memRequest{data: cloneCapturedRequest(req), createdAt: now}

	return &RequestMeta{CreatedAt: now}, nil
}

// GetRequest implements [RequestStorage].
func (s *Memory) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	sess, sessOk := s.sessions.Get(sID)
	if !sessOk {
		return nil, ErrSessionNotFound
	}

	sess.mu.RLock()
	defer sess.mu.RUnlock()

	if sess.isExpired(s.timeNow()) {
		return nil, ErrSessionNotFound
	}

	r, reqOk := sess.requests[rID]
	if !reqOk {
		return nil, ErrRequestNotFound
	}

	return new(Request{Data: cloneCapturedRequest(r.data), Meta: RequestMeta{CreatedAt: r.createdAt}}), nil
}

// GetRequests implements [RequestStorage].
func (s *Memory) GetRequests(ctx context.Context, sID string, _ *error) (iter.Seq2[string, Request], error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	sess.mu.RLock()

	if sess.isExpired(s.timeNow()) {
		sess.mu.RUnlock()

		return nil, ErrSessionNotFound
	}

	entries := make([]sortEntry, 0, len(sess.requests))

	for id, r := range sess.requests {
		entries = append(entries, sortEntry{id: id, createdAt: r.createdAt})
	}

	sess.mu.RUnlock()

	// sort requests newest-first by creation time
	slices.SortFunc(entries, func(a, b sortEntry) int { return b.createdAt.Compare(a.createdAt) })

	return func(yield func(string, Request) bool) {
		for _, e := range entries {
			sess.mu.RLock()

			r, exists := sess.requests[e.id]
			if !exists {
				sess.mu.RUnlock()

				continue // request was deleted between snapshot and iteration
			}

			req := Request{Data: cloneCapturedRequest(r.data), Meta: RequestMeta{CreatedAt: r.createdAt}}

			sess.mu.RUnlock()

			if !yield(e.id, req) {
				return
			}
		}
	}, nil
}

// DeleteRequest implements [RequestStorage].
func (s *Memory) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	if _, reqOk := sess.requests[rID]; !reqOk {
		return ErrRequestNotFound
	}

	delete(sess.requests, rID)

	return nil
}

// DeleteAllRequests implements [RequestStorage].
func (s *Memory) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	sess, ok := s.sessions.Get(sID)
	if !ok {
		return ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	clear(sess.requests)

	return nil
}

// Close implements [io.Closer]. It stops the cleanup goroutine, waits for it to exit, and marks
// the storage as closed. All subsequent method calls return [ErrClosed]. Safe to call more than once.
func (s *Memory) Close() error {
	s.closeOnce.Do(func() { close(s.closeSignalCh) })
	<-s.closedCh

	return nil
}

// --------------------------------------------------------------------------------------------------------------------

// cloneSessionResponse creates a deep copy of the SessionResponse.
func cloneSessionResponse(r SessionResponse) SessionResponse {
	var headers []ResponseHeader
	if len(r.Headers) > 0 {
		headers = make([]ResponseHeader, len(r.Headers))
		copy(headers, r.Headers)
	}

	var body []byte
	if len(r.Body) > 0 {
		body = make([]byte, len(r.Body))
		copy(body, r.Body)
	}

	return SessionResponse{
		Code:    r.Code,
		Headers: headers,
		Body:    body,
		Delay:   r.Delay,
	}
}

// cloneCapturedRequest creates a deep copy of the CapturedRequest.
func cloneCapturedRequest(c CapturedRequest) CapturedRequest {
	var headers []RequestHeader
	if len(c.Headers) > 0 {
		headers = make([]RequestHeader, len(c.Headers))
		copy(headers, c.Headers)
	}

	var body []byte
	if len(c.Body) > 0 {
		body = make([]byte, len(c.Body))
		copy(body, c.Body)
	}

	return CapturedRequest{
		ClientAddr: c.ClientAddr,
		Method:     c.Method,
		Body:       body,
		Headers:    headers,
		URL:        c.URL,
	}
}

// memSession holds all data for a single session. Its fields are protected by mu.
type memSession struct {
	mu        sync.RWMutex
	response  SessionResponse
	createdAt time.Time
	expiresAt time.Time // zero value means no expiration
	requests  map[string]*memRequest
}

// isExpired reports whether the session has passed its expiration deadline.
// The caller must hold at least sess.mu.RLock.
func (sess *memSession) isExpired(now time.Time) bool {
	return !sess.expiresAt.IsZero() && sess.expiresAt.Before(now)
}

// toSession returns a deep copy of the session as a [Session] value.
// The caller must hold at least sess.mu.RLock.
func (sess *memSession) toSession() Session {
	return Session{
		Response: cloneSessionResponse(sess.response),
		Meta:     SessionMeta{CreatedAt: sess.createdAt, ExpiresAt: sess.expiresAt},
	}
}

// memSessionMap is a mutex-guarded map of sessions.
type memSessionMap struct {
	mu   sync.RWMutex
	data map[string]*memSession
}

// Get returns the session stored under id, or (nil, false) if absent.
// The returned pointer is only safe to dereference under session-level locking.
func (m *memSessionMap) Get(id string) (*memSession, bool) {
	m.mu.RLock()
	s, ok := m.data[id]
	m.mu.RUnlock()

	return s, ok
}

type memRequest struct {
	data      CapturedRequest
	createdAt time.Time
}

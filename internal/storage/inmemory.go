package storage

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type inmemorySession struct {
	uuid        string
	content     []byte
	code        uint16
	contentType string
	delay       time.Duration
	createdAt   time.Time
	requests    map[string]*inmemoryRequest // key is request UUID

	expiresAtNano int64
}

func (s *inmemorySession) UUID() string         { return s.uuid }        // UUID unique session ID.
func (s *inmemorySession) Content() []byte      { return s.content }     // Content session server response content.
func (s *inmemorySession) Code() uint16         { return s.code }        // Code default server response code.
func (s *inmemorySession) ContentType() string  { return s.contentType } // ContentType response content type.
func (s *inmemorySession) Delay() time.Duration { return s.delay }       // Delay before response sending.
func (s *inmemorySession) CreatedAt() time.Time { return s.createdAt }   // CreatedAt creation time.

type inmemoryRequest struct {
	uuid       string
	clientAddr string
	method     string
	content    []byte
	headers    map[string]string
	uri        string
	createdAt  time.Time
}

func (r *inmemoryRequest) UUID() string               { return r.uuid }       // UUID returns unique request ID.
func (r *inmemoryRequest) ClientAddr() string         { return r.clientAddr } // ClientAddr client hostname or IP.
func (r *inmemoryRequest) Method() string             { return r.method }     // Method HTTP method name.
func (r *inmemoryRequest) Content() []byte            { return r.content }    // Content request body (payload).
func (r *inmemoryRequest) Headers() map[string]string { return r.headers }    // Headers HTTP request headers.
func (r *inmemoryRequest) URI() string                { return r.uri }        // URI Uniform Resource Identifier.
func (r *inmemoryRequest) CreatedAt() time.Time       { return r.createdAt }  // CreatedAt creation time.

var ErrClosed = errors.New("closed")

type InMemory struct {
	sessionTTL  time.Duration
	maxRequests uint16

	cleanupInterval time.Duration

	storageMu sync.RWMutex
	storage   map[string]*inmemorySession // key is session UUID

	close    chan struct{}
	closedMu sync.RWMutex
	closed   bool
}

const defaultInMemoryCleanupInterval = time.Second // default cleanup interval

// NewInMemory creates inmemory storage.
func NewInMemory(sessionTTL time.Duration, maxRequests uint16, cleanup ...time.Duration) *InMemory {
	ci := defaultInMemoryCleanupInterval

	if len(cleanup) > 0 {
		ci = cleanup[0]
	}

	s := &InMemory{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		cleanupInterval: ci,
		storage:         make(map[string]*inmemorySession),
		close:           make(chan struct{}, 1),
	}
	go s.cleanup()

	return s
}

func (s *InMemory) cleanup() {
	defer close(s.close)

	timer := time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.close:
			s.storageMu.Lock()
			for id := range s.storage {
				delete(s.storage, id)
			}
			s.storageMu.Unlock()

			return

		case <-timer.C:
			s.storageMu.Lock()
			var now = time.Now().UnixNano()

			for id, session := range s.storage {
				if now > session.expiresAtNano {
					delete(s.storage, id)
				}
			}
			s.storageMu.Unlock()

			timer.Reset(s.cleanupInterval)
		}
	}
}

func (s *InMemory) isClosed() (closed bool) {
	s.closedMu.RLock()
	closed = s.closed
	s.closedMu.RUnlock()

	return
}

// Close current storage with data invalidation.
func (s *InMemory) Close() error {
	if s.isClosed() {
		return ErrClosed
	}

	s.closedMu.Lock()
	s.closed = true
	s.closedMu.Unlock()

	s.close <- struct{}{}

	return nil
}

// GetSession returns session data.
func (s *InMemory) GetSession(uuid string) (Session, error) {
	if s.isClosed() {
		return nil, ErrClosed
	}

	s.storageMu.RLock()
	session, ok := s.storage[uuid]
	s.storageMu.RUnlock()

	if ok {
		// session has been expired?
		if time.Now().UnixNano() > session.expiresAtNano {
			s.storageMu.Lock()
			delete(s.storage, uuid)
			s.storageMu.Unlock()

			return nil, nil // session has been expired (not found)
		}

		return session, nil
	}

	return nil, nil // not found
}

// CreateSession creates new session in storage using passed data.
func (s *InMemory) CreateSession(content []byte, code uint16, contentType string, delay time.Duration, sessionUUID ...string) (string, error) { //nolint:lll
	if s.isClosed() {
		return "", ErrClosed
	}

	var id string

	if len(sessionUUID) == 1 && IsValidUUID(sessionUUID[0]) {
		id = sessionUUID[0]
	} else {
		id = NewUUID()
	}

	now := time.Now()

	s.storageMu.Lock()
	s.storage[id] = &inmemorySession{
		uuid:          id,
		content:       content,
		code:          code,
		contentType:   contentType,
		delay:         delay,
		createdAt:     now,
		requests:      make(map[string]*inmemoryRequest, s.maxRequests),
		expiresAtNano: now.UnixNano() + s.sessionTTL.Nanoseconds(),
	}
	s.storageMu.Unlock()

	return id, nil
}

// DeleteSession deletes session with passed UUID.
func (s *InMemory) DeleteSession(uuid string) (bool, error) {
	session, err := s.GetSession(uuid)
	if err != nil {
		return false, err
	}

	if session != nil {
		s.storageMu.Lock()
		delete(s.storage, uuid)
		s.storageMu.Unlock()

		return true, nil // found and deleted
	}

	return false, nil // session was not found
}

// DeleteRequests deletes stored requests for session with passed UUID.
func (s *InMemory) DeleteRequests(uuid string) (bool, error) {
	session, err := s.GetSession(uuid)
	if err != nil {
		return false, err
	}

	if session != nil {
		s.storageMu.Lock()
		defer s.storageMu.Unlock()

		if len(s.storage[uuid].requests) == 0 {
			return false, nil // nothing to delete
		}

		for id := range s.storage[uuid].requests {
			delete(s.storage[uuid].requests, id)
		}

		return true, nil // requests deleted
	}

	return false, nil // session was not found
}

// CreateRequest creates new request in storage using passed data and updates expiration time for session and all
// stored requests for the session.
func (s *InMemory) CreateRequest(sessionUUID, clientAddr, method, uri string, content []byte, headers map[string]string) (string, error) { //nolint:lll
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return "", err
	}

	if session != nil {
		s.storageMu.Lock()
		defer s.storageMu.Unlock()

		now := time.Now()
		id := NewUUID()

		// append new request
		s.storage[sessionUUID].requests[id] = &inmemoryRequest{
			uuid:       id,
			clientAddr: clientAddr,
			method:     method,
			content:    content,
			headers:    headers,
			uri:        uri,
			createdAt:  now,
		}

		// update session TTL
		s.storage[sessionUUID].expiresAtNano = now.UnixNano() + s.sessionTTL.Nanoseconds()

		// limit stored requests count
		if rl := len(s.storage[sessionUUID].requests); rl > int(s.maxRequests) {
			type rq struct {
				id string
				ts int64
			}

			allReq := make([]rq, 0, rl)

			for k := range s.storage[sessionUUID].requests {
				allReq = append(allReq, rq{k, s.storage[sessionUUID].requests[k].createdAt.UnixNano()})
			}

			sort.Slice(allReq, func(i, j int) bool { return allReq[i].ts > allReq[j].ts })

			for i, plan := 0, allReq[int(s.maxRequests):]; i < len(plan); i++ {
				delete(s.storage[sessionUUID].requests, plan[i].id)
			}
		}

		return id, nil // request added
	}

	return "", nil // session was not found
}

// GetRequest returns request data.
func (s *InMemory) GetRequest(sessionUUID, requestUUID string) (Request, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		s.storageMu.RLock()
		defer s.storageMu.RUnlock()

		if _, reqOk := s.storage[sessionUUID].requests[requestUUID]; reqOk {
			return s.storage[sessionUUID].requests[requestUUID], nil
		}

		return nil, nil // request was not found
	}

	return nil, nil // session was not found
}

// GetAllRequests returns all request as a slice of structures.
func (s *InMemory) GetAllRequests(sessionUUID string) ([]Request, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		s.storageMu.RLock()
		defer s.storageMu.RUnlock()

		if len(s.storage[sessionUUID].requests) == 0 {
			return nil, nil // no requests
		}

		result := make([]Request, 0, len(s.storage[sessionUUID].requests))
		for id := range s.storage[sessionUUID].requests {
			result = append(result, s.storage[sessionUUID].requests[id])
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].(*inmemoryRequest).createdAt.UnixNano() < result[j].(*inmemoryRequest).createdAt.UnixNano()
		})

		return result, nil
	}

	return nil, nil // session was not found
}

// DeleteRequest deletes stored request with passed session and request UUIDs.
func (s *InMemory) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return false, err
	}

	if session != nil {
		s.storageMu.Lock()
		defer s.storageMu.Unlock()

		if _, ok := s.storage[sessionUUID].requests[requestUUID]; ok {
			delete(s.storage[sessionUUID].requests, requestUUID)

			return true, nil // deleted
		}

		return false, nil // request was not found
	}

	return false, nil // session was not found
}

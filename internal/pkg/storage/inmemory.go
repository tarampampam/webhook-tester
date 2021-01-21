package storage

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type inmemorySession struct {
	uuid        string
	content     string
	code        uint16
	contentType string
	delay       time.Duration
	createdAt   time.Time
	requests    map[string]*inmemoryRequest // key is request UUID

	expiresAtNano int64
}

func (s *inmemorySession) UUID() string         { return s.uuid }        // UUID unique session ID.
func (s *inmemorySession) Content() string      { return s.content }     // Content session server content.
func (s *inmemorySession) Code() uint16         { return s.code }        // Code default server response code.
func (s *inmemorySession) ContentType() string  { return s.contentType } // ContentType response content type.
func (s *inmemorySession) Delay() time.Duration { return s.delay }       // Delay before response sending.
func (s *inmemorySession) CreatedAt() time.Time { return s.createdAt }   // CreatedAt creation time.

type inmemoryRequest struct {
	uuid       string
	clientAddr string
	method     string
	content    string
	headers    map[string]string
	uri        string
	createdAt  time.Time
}

func (r *inmemoryRequest) UUID() string               { return r.uuid }       // UUID returns unique request ID.
func (r *inmemoryRequest) ClientAddr() string         { return r.clientAddr } // ClientAddr client hostname or IP.
func (r *inmemoryRequest) Method() string             { return r.method }     // Method HTTP method name.
func (r *inmemoryRequest) Content() string            { return r.content }    // Content request body (payload).
func (r *inmemoryRequest) Headers() map[string]string { return r.headers }    // Headers HTTP request headers.
func (r *inmemoryRequest) URI() string                { return r.uri }        // URI Uniform Resource Identifier.
func (r *inmemoryRequest) CreatedAt() time.Time       { return r.createdAt }  // CreatedAt creation time.

var ErrClosed = errors.New("closed")

type InMemoryStorage struct {
	sessionTTL  time.Duration
	maxRequests uint16

	cleanupInterval time.Duration

	storMu sync.RWMutex
	stor   map[string]*inmemorySession // key is session UUID

	close    chan struct{}
	closedMu sync.RWMutex
	closed   bool
}

func NewInMemoryStorage(sessionTTL time.Duration, maxRequests uint16, cleanupInterval time.Duration) *InMemoryStorage {
	s := &InMemoryStorage{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		cleanupInterval: cleanupInterval,
		stor:            make(map[string]*inmemorySession),
		close:           make(chan struct{}, 1),
	}
	go s.cleanup()

	return s
}

func (s *InMemoryStorage) cleanup() {
	defer close(s.close)

	timer := time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.close:
			s.storMu.Lock()
			for id := range s.stor {
				delete(s.stor, id)
			}
			s.storMu.Unlock()

			return

		case <-timer.C:
			s.storMu.Lock()
			var now = time.Now().UnixNano()

			for id, session := range s.stor {
				if now > session.expiresAtNano {
					delete(s.stor, id)
				}
			}
			s.storMu.Unlock()

			timer.Reset(s.cleanupInterval)
		}
	}
}

func (s *InMemoryStorage) isClosed() bool {
	s.closedMu.RLock()
	defer s.closedMu.RUnlock()

	return s.closed
}

// Close current storage with data invalidation.
func (s *InMemoryStorage) Close() error {
	if s.isClosed() {
		return ErrClosed
	}

	s.closedMu.Lock()
	s.closed = true
	s.closedMu.Unlock()

	s.close <- struct{}{}

	return nil
}

func (s *InMemoryStorage) newUUID() string { return uuid.New().String() }

// GetSession returns session data.
func (s *InMemoryStorage) GetSession(uuid string) (Session, error) {
	if s.isClosed() {
		return nil, ErrClosed
	}

	s.storMu.RLock()
	session, ok := s.stor[uuid]
	s.storMu.RUnlock()

	if ok {
		// session has been expired?
		if time.Now().UnixNano() > session.expiresAtNano {
			s.storMu.Lock()
			delete(s.stor, uuid)
			s.storMu.Unlock()

			return nil, nil // session has been expired (not found)
		}

		return session, nil
	}

	return nil, nil // not found
}

// CreateSession creates new session in storage using passed data.
func (s *InMemoryStorage) CreateSession(content string, code uint16, contentType string, delay time.Duration) (string, error) { //nolint:lll
	if s.isClosed() {
		return "", ErrClosed
	}

	id := s.newUUID()
	now := time.Now()

	s.storMu.Lock()
	s.stor[id] = &inmemorySession{
		uuid:          id,
		content:       content,
		code:          code,
		contentType:   contentType,
		delay:         delay,
		createdAt:     now,
		requests:      make(map[string]*inmemoryRequest, s.maxRequests),
		expiresAtNano: now.UnixNano() + s.sessionTTL.Nanoseconds(),
	}
	s.storMu.Unlock()

	return id, nil
}

// DeleteSession deletes session with passed UUID.
func (s *InMemoryStorage) DeleteSession(uuid string) (bool, error) {
	session, err := s.GetSession(uuid)
	if err != nil {
		return false, err
	}

	if session != nil {
		s.storMu.Lock()
		delete(s.stor, uuid)
		s.storMu.Unlock()

		return true, nil // found and deleted
	}

	return false, nil // session was not found
}

// DeleteRequests deletes stored requests for session with passed UUID.
func (s *InMemoryStorage) DeleteRequests(uuid string) (bool, error) {
	session, err := s.GetSession(uuid)
	if err != nil {
		return false, err
	}

	if session != nil {
		s.storMu.Lock()
		defer s.storMu.Unlock()

		if len(s.stor[uuid].requests) == 0 {
			return false, nil // nothing to delete
		}

		for id := range s.stor[uuid].requests {
			delete(s.stor[uuid].requests, id)
		}

		return true, nil // requests deleted
	}

	return false, nil // session was not found
}

// CreateRequest creates new request in storage using passed data.
func (s *InMemoryStorage) CreateRequest(sessionUUID, clientAddr, method, content, uri string, headers map[string]string) (string, error) { //nolint:lll
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return "", err
	}

	if session != nil {
		s.storMu.Lock()
		defer s.storMu.Unlock()

		now := time.Now()
		id := s.newUUID()

		// append new request
		s.stor[sessionUUID].requests[id] = &inmemoryRequest{
			uuid:       id,
			clientAddr: clientAddr,
			method:     method,
			content:    content,
			headers:    headers,
			uri:        uri,
			createdAt:  now,
		}

		// update session TTL
		s.stor[sessionUUID].expiresAtNano = now.UnixNano() + s.sessionTTL.Nanoseconds()

		// limit stored requests count
		if rl := len(s.stor[sessionUUID].requests); rl > int(s.maxRequests) {
			type rq struct {
				id string
				ts int64
			}

			allReq := make([]rq, 0, rl)

			for k := range s.stor[sessionUUID].requests {
				allReq = append(allReq, rq{k, s.stor[sessionUUID].requests[k].createdAt.UnixNano()})
			}

			sort.Slice(allReq, func(i, j int) bool { return allReq[i].ts > allReq[j].ts })

			for i, plan := 0, allReq[int(s.maxRequests):]; i < len(plan); i++ {
				delete(s.stor[sessionUUID].requests, plan[i].id)
			}
		}

		return id, nil // request added
	}

	return "", nil // session was not found
}

// GetRequest returns request data.
func (s *InMemoryStorage) GetRequest(sessionUUID, requestUUID string) (Request, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		if _, reqOk := s.stor[sessionUUID].requests[requestUUID]; reqOk {
			return s.stor[sessionUUID].requests[requestUUID], nil
		}

		return nil, nil // request was not found
	}

	return nil, nil // session was not found
}

// GetAllRequests returns all request as a slice of structures.
func (s *InMemoryStorage) GetAllRequests(sessionUUID string) ([]Request, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		if len(s.stor[sessionUUID].requests) == 0 {
			return nil, nil // no requests
		}

		result := make([]Request, 0, len(s.stor[sessionUUID].requests))
		for id := range s.stor[sessionUUID].requests {
			result = append(result, s.stor[sessionUUID].requests[id])
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].(*inmemoryRequest).createdAt.UnixNano() < result[j].(*inmemoryRequest).createdAt.UnixNano()
		})

		return result, nil
	}

	return nil, nil // session was not found
}

// DeleteRequest deletes stored request with passed session and request UUIDs.
func (s *InMemoryStorage) DeleteRequest(sessionUUID, requestUUID string) (bool, error) {
	session, err := s.GetSession(sessionUUID)
	if err != nil {
		return false, err
	}

	if session != nil {
		if _, ok := s.stor[sessionUUID].requests[requestUUID]; ok {
			delete(s.stor[sessionUUID].requests, requestUUID)

			return true, nil // deleted
		}

		return false, nil // request was not found
	}

	return false, nil // session was not found
}

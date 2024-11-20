package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

type SQLite struct { // TODO: use transactions?
	sessionTTL      time.Duration
	maxRequests     uint32
	db              *sql.DB
	cleanupInterval time.Duration

	close  chan struct{}
	closed atomic.Bool
}

var ( // ensure interface implementation
	_ Storage   = (*SQLite)(nil)
	_ io.Closer = (*SQLite)(nil)
)

type SQLiteOption func(*SQLite)

func WithSQLiteCleanupInterval(v time.Duration) SQLiteOption {
	return func(s *SQLite) { s.cleanupInterval = v }
}

func NewSQLite(
	db *sql.DB,
	sessionTTL time.Duration,
	maxRequests uint32,
	opts ...SQLiteOption,
) (*SQLite, error) {
	var s = SQLite{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		db:              db,
		close:           make(chan struct{}),
		cleanupInterval: time.Second, // default cleanup interval
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.cleanupInterval > time.Duration(0) {
		go s.cleanup(context.Background()) // start cleanup goroutine
	}

	return &s, nil
}

// newID generates a new (unique) ID.
func (*SQLite) newID() string { return uuid.New().String() }

func (s *SQLite) Init(ctx context.Context) error {
	for _, query := range []string{
		`CREATE TABLE IF NOT EXISTS sessions (
				id           VARCHAR(36)      NOT NULL,
				code         UNSIGNED INTEGER NOT NULL,
				delay_millis UNSIGNED INTEGER NOT NULL,
				body         BLOB             NULL,
				created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
				expires_at   DATETIME         NOT NULL,
				CONSTRAINT chk_sessions_response_code CHECK (code >= 0 AND code <= 65535),
				CONSTRAINT chk_sessions_expires_at    CHECK (expires_at >= CURRENT_TIMESTAMP)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS unq_sessions_id ON sessions(id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS response_headers (
				session_id VARCHAR(36)   NOT NULL,
				name       VARCHAR(1024) NOT NULL,
				value      TEXT          NOT NULL,
				FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_response_headers_session_id ON response_headers(session_id)`,
		`CREATE TABLE IF NOT EXISTS requests (
				id             VARCHAR(36)   NOT NULL,
				session_id     VARCHAR(36)   NOT NULL,
				method         VARCHAR(10)   NOT NULL,
				client_address VARCHAR(39)   NOT NULL,
				url            VARCHAR(4096) NOT NULL,
				payload        BLOB          NULL,
				created_at     DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS unq_requests_id ON requests(id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_session_id ON requests(session_id)`,
		`CREATE TABLE IF NOT EXISTS request_headers (
				request_id VARCHAR(36)   NOT NULL,
				name       VARCHAR(1024) NOT NULL,
				value      TEXT          NOT NULL,
				FOREIGN KEY(request_id) REFERENCES requests(id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_request_headers_request_id ON request_headers(request_id)`,
	} {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLite) cleanup(ctx context.Context) {
	var timer = time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.close: // close signal received
			return
		case <-timer.C:
			_, _ = s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
			timer.Reset(s.cleanupInterval)
		}
	}
}

func (s *SQLite) isSessionExists(ctx context.Context, sID string) (exists bool, _ error) {
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM sessions WHERE id = ? AND expires_at >= CURRENT_TIMESTAMP)`,
		sID,
	).Scan(&exists); err != nil {
		return false, err
	}

	return
}

func (s *SQLite) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) {
	if err := ctx.Err(); err != nil {
		return "", err // context is done
	} else if s.closed.Load() {
		return "", ErrClosed // storage is closed
	}

	if len(id) > 0 { // use the specified ID
		if len(id[0]) == 0 {
			return "", errors.New("empty session ID")
		}

		sID = id[0]

		// check if the session with the specified ID already exists
		if exists, err := s.isSessionExists(ctx, sID); err != nil {
			return "", err
		} else if exists {
			return "", fmt.Errorf("session %s already exists", sID)
		}
	} else {
		sID = s.newID() // generate a new ID
	}

	{ // store the session
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO sessions (id, code, delay_millis, body, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
			sID, session.Code, session.Delay.Milliseconds(), session.ResponseBody, time.Now().Add(s.sessionTTL),
		)
		if err != nil {
			return "", err
		}
	}

	{ // store headers using insert batch
		var args, values = make([]any, 0, len(session.Headers)*3), make([]string, 0, len(session.Headers)*3) //nolint:mnd

		for _, h := range session.Headers {
			args = append(args, sID, h.Name, h.Value)
			values = append(values, "(?, ?, ?)")
		}

		if len(args) > 0 {
			if _, err := s.db.ExecContext(
				ctx,
				`INSERT INTO response_headers (session_id, name, value) VALUES `+strings.Join(values, ", "), //nolint:gosec
				args...,
			); err != nil {
				return "", err
			}
		}
	}

	return sID, nil
}

func (s *SQLite) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	var (
		session  Session
		createAt time.Time
		delay    int64
	)

	if err := s.db.QueryRowContext(
		ctx,
		`SELECT code, delay_millis, body, created_at, expires_at FROM sessions WHERE id = ?`,
		sID,
	).Scan(&session.Code, &delay, &session.ResponseBody, &createAt, &session.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	session.CreatedAtUnixMilli = createAt.UnixMilli()
	session.Delay = time.Millisecond * time.Duration(delay)

	if session.ExpiresAt.Before(time.Now()) {
		_, _ = s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", sID)

		return nil, ErrSessionNotFound // session has been expired
	}

	// load session headers
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT name, value FROM response_headers WHERE session_id = ?`,
		sID,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h HttpHeader

		if err = rows.Scan(&h.Name, &h.Value); err != nil {
			return nil, err
		}

		session.Headers = append(session.Headers, h)
	}

	return &session, rows.Err()
}

func (s *SQLite) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	if _, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions SET expires_at = expires_at + ? WHERE id = ?`,
		howMuch, sID,
	); err != nil {
		return err
	}

	return nil
}

func (s *SQLite) DeleteSession(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	if res, execErr := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", sID); execErr != nil {
		if errors.Is(execErr, sql.ErrNoRows) {
			return ErrSessionNotFound
		}

		return execErr
	} else if affected, err := res.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (s *SQLite) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) { //nolint:funlen
	if err := ctx.Err(); err != nil {
		return "", err // context is done
	} else if s.closed.Load() {
		return "", ErrClosed // storage is closed
	}

	if exists, err := s.isSessionExists(ctx, sID); err != nil {
		return "", err
	} else if !exists {
		return "", ErrSessionNotFound
	}

	rID = s.newID() // generate a new ID

	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO requests (id, session_id, method, client_address, url, payload)
		VALUES (?, ?, ?, ?, ?, ?)`,
		rID, sID, r.Method, r.ClientAddr, r.URL, r.Body,
	); err != nil {
		return "", err
	}

	// store request headers using insert batch
	var args, values = make([]any, 0, len(r.Headers)*3), make([]string, 0, len(r.Headers)*3) //nolint:mnd

	for _, h := range r.Headers {
		args = append(args, rID, h.Name, h.Value)
		values = append(values, "(?, ?, ?)")
	}

	if len(args) > 0 {
		if _, err := s.db.ExecContext(
			ctx,
			`INSERT INTO request_headers (request_id, name, value) VALUES `+strings.Join(values, ", "), //nolint:gosec
			args...,
		); err != nil {
			return "", err
		}
	}

	{ // limit stored requests count
		var count int

		if err := s.db.QueryRowContext(
			ctx,
			`SELECT COUNT(id) FROM requests WHERE session_id = ?`,
			sID,
		).Scan(&count); err != nil {
			return "", err
		}

		if count > int(s.maxRequests) {
			// delete all requests from the requests table that are not in the last N requests, ordered by creation time
			if _, err := s.db.ExecContext(
				ctx,
				`DELETE FROM requests WHERE session_id = ? AND id NOT IN (
					SELECT id FROM requests WHERE session_id = ? ORDER BY created_at DESC LIMIT ?
				)`,
				sID, sID, s.maxRequests,
			); err != nil {
				return "", err
			}
		}
	}

	return rID, nil
}

func (s *SQLite) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	// check the session existence
	if exists, err := s.isSessionExists(ctx, sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	var (
		request   Request
		createdAt time.Time
	)

	if err := s.db.QueryRowContext(
		ctx,
		`SELECT method, client_address, url, payload, created_at FROM requests WHERE id = ? AND session_id = ?`,
		rID, sID,
	).Scan(&request.Method, &request.ClientAddr, &request.URL, &request.Body, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}

		return nil, err
	}

	request.CreatedAtUnixMilli = createdAt.UnixMilli()

	// load request headers
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT name, value FROM request_headers WHERE request_id = ?`,
		rID,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h HttpHeader

		if err = rows.Scan(&h.Name, &h.Value); err != nil {
			return nil, err
		}

		request.Headers = append(request.Headers, h)
	}

	return &request, rows.Err()
}

func (s *SQLite) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) { //nolint:funlen
	if err := ctx.Err(); err != nil {
		return nil, err // context is done
	} else if s.closed.Load() {
		return nil, ErrClosed // storage is closed
	}

	// check the session existence
	if exists, err := s.isSessionExists(ctx, sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	// read all stored request IDs
	rRows, rErr := s.db.QueryContext(
		ctx,
		`SELECT id, method, client_address, url, payload, created_at FROM requests WHERE session_id = ?`,
		sID,
	)
	if rErr != nil {
		return nil, rErr
	}

	defer func() { _ = rRows.Close() }()

	var all = make(map[string]Request)

	for rRows.Next() {
		var (
			request   Request
			rID       string
			createdAt time.Time
		)

		if sErr := rRows.Scan(
			&rID,
			&request.Method,
			&request.ClientAddr,
			&request.URL,
			&request.Body,
			&createdAt,
		); sErr != nil {
			return nil, sErr
		}

		request.CreatedAtUnixMilli = createdAt.UnixMilli()

		all[rID] = request
	}

	if err := rRows.Err(); err != nil {
		return nil, err
	}

	_ = rRows.Close() // close the rows asap

	// load all the request headers in a single query
	hRows, hErr := s.db.QueryContext(
		ctx,
		`SELECT request_id, name, value FROM request_headers WHERE request_id IN (
			SELECT id FROM requests WHERE session_id = ?
		)`,
		sID,
	)
	if hErr != nil {
		return nil, hErr
	}

	defer func() { _ = hRows.Close() }()

	for hRows.Next() {
		var (
			rID string
			h   HttpHeader
		)

		if err := hRows.Scan(&rID, &h.Name, &h.Value); err != nil {
			return nil, err
		}

		if req, ok := all[rID]; ok {
			req.Headers = append(req.Headers, h)
			all[rID] = req
		}
	}

	return all, hRows.Err()
}

func (s *SQLite) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	// check the session existence first
	if exists, err := s.isSessionExists(ctx, sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	if res, execErr := s.db.ExecContext(
		ctx,
		"DELETE FROM requests WHERE id = ? AND session_id = ?",
		rID, sID,
	); execErr != nil {
		if errors.Is(execErr, sql.ErrNoRows) {
			return ErrRequestNotFound
		}

		return execErr
	} else if affected, err := res.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return ErrRequestNotFound
	}

	return nil
}

func (s *SQLite) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	// check the session existence
	if exists, err := s.isSessionExists(ctx, sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// delete all requests
	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM requests WHERE session_id = ?`,
		sID,
	); err != nil {
		return err
	}

	return nil
}

func (s *SQLite) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		close(s.close)

		return nil
	}

	return ErrClosed
}

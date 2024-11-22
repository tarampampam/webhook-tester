package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

//go:embed sqlite_migrations/*.sql
var sqliteMigrations embed.FS

type (
	SQLite struct { // TODO: use transactions?
		sessionTTL      time.Duration
		maxRequests     uint32
		cleanupInterval time.Duration

		// every operation with the database should be wrapped in a transaction to avoid "database is locked" errors,
		// which are common for SQLite when multiple operations are performed simultaneously. that's why we don't use
		// the db directly
		newTx sqliteNewTx

		close  chan struct{}
		closed atomic.Bool
	}

	sqliteNewTx func(_ context.Context, readOnly bool) (*sql.Tx, func(commit bool) error, error)
)

var ( // ensure interface implementation
	_ Storage   = (*SQLite)(nil)
	_ io.Closer = (*SQLite)(nil)
)

// SQLiteOption is a functional option type for the SQLite storage.
type SQLiteOption func(*SQLite)

// WithSQLiteCleanupInterval sets the cleanup interval for the SQLite storage.
func WithSQLiteCleanupInterval(v time.Duration) SQLiteOption {
	return func(s *SQLite) { s.cleanupInterval = v }
}

// NewSQLite creates a new SQLite storage instance.
//
// You need to call the [SQLite.Migrate] method to apply the migrations before using the storage.
// [SQLite.Close] should be called once you're done with the storage to stop the cleanup goroutine and make storage
// inaccessible.
func NewSQLite(db *sql.DB, sessionTTL time.Duration, maxRequests uint32, opts ...SQLiteOption) *SQLite {
	var s = SQLite{
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		cleanupInterval: time.Second, // default cleanup interval
		close:           make(chan struct{}),
	}

	s.newTx = s.newTxCreator(db) // set the transaction creator function

	for _, opt := range opts {
		opt(&s)
	}

	if s.cleanupInterval > time.Duration(0) {
		go s.cleanup(context.Background()) // start cleanup goroutine
	}

	return &s
}

// newTxCreator creates a new transaction creator function for the SQLite storage. Every transaction is executed with
// the serializable isolation level.
//
// Once you're done with the transaction, you should call the returned commit function to commit or rollback the
// transaction. It's safe to call it multiple times, but only the first call will be used.
func (*SQLite) newTxCreator(db *sql.DB) sqliteNewTx {
	// protects SQLite from concurrent access, to avoid "database is locked" errors
	var mu sync.RWMutex

	return func(ctx context.Context, readOnly bool) (*sql.Tx, func(bool) error, error) {
		// lock the database
		if readOnly {
			mu.RLock()
		} else {
			mu.Lock()
		}

		// https://www.sqlite.org/isolation.html
		var tx, txErr = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: readOnly})
		if txErr != nil {
			return nil, func(bool) error { return txErr }, fmt.Errorf("failed to start a new transaction: %w", txErr)
		}

		var once sync.Once

		return tx, func(commit bool) (err error) {
			once.Do(func() {
				if commit {
					if err = tx.Commit(); err != nil {
						err = fmt.Errorf("failed to commit the transaction: %w", err)
					}
				} else {
					if err = tx.Rollback(); err != nil {
						err = fmt.Errorf("failed to rollback the transaction: %w", err)
					}
				}

				// unlock the database
				if readOnly {
					mu.RUnlock()
				} else {
					mu.Unlock()
				}
			})

			return
		}, nil
	}
}

// newID generates a new (unique) ID.
func (*SQLite) newID() string { return uuid.New().String() }

// Migrate the SQLite storage by applying the migrations. Can be called multiple times, it's safe.
func (s *SQLite) Migrate(ctx context.Context) error {
	return fs.WalkDir(sqliteMigrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if d.IsDir() {
			return nil
		}

		var file, fErr = sqliteMigrations.Open(path) // open the file
		if fErr != nil {
			return fErr
		}

		defer func() { _ = file.Close() }()

		var data, rErr = io.ReadAll(file) // read the file content
		if rErr != nil {
			return rErr
		}

		tx, commit, txErr := s.newTx(ctx, false)
		if txErr != nil {
			return txErr
		}

		defer func() { _ = commit(false) }() // rollback the transaction in case of an error

		if _, eErr := tx.ExecContext(ctx, string(data)); eErr != nil { // execute the query from the file
			return eErr
		}

		if cErr := commit(true); cErr != nil { // commit the transaction
			return cErr
		}

		return nil
	})
}

// cleanup is a goroutine that cleans up the expired sessions.
func (s *SQLite) cleanup(ctx context.Context) {
	var timer = time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.close: // close signal received
			return
		case <-timer.C:
			if tx, commit, txErr := s.newTx(ctx, false); txErr == nil {
				_, err := tx.ExecContext(ctx, "DELETE FROM `sessions` WHERE `expires_at_millis` < ?", time.Now().UnixMilli())
				_ = commit(err == nil)
			}

			timer.Reset(s.cleanupInterval)
		}
	}
}

// isSessionExists checks if the session with the specified ID exists and is not expired.
func (*SQLite) isSessionExists(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, sID string) (exists bool, _ error) {
	if err := db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM `sessions` WHERE `id` = ? AND `expires_at_millis` >= ?)",
		sID, time.Now().UnixMilli(),
	).Scan(&exists); err != nil {
		return false, err
	}

	return
}

// isOpenAndNotDone checks if the storage is open and the context is not done.
func (*SQLite) isOpenAndNotDone(ctx context.Context, s *SQLite) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	return nil
}

func (s *SQLite) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return "", err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return "", txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	if len(id) > 0 { // use the specified ID
		if len(id[0]) == 0 {
			return "", errors.New("empty session ID")
		}

		sID = id[0]

		// check if the session with the specified ID already exists
		if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
			return "", err
		} else if exists {
			return "", fmt.Errorf("session %s already exists", sID)
		}
	} else {
		sID = s.newID() // generate a new ID
	}

	{ // store the session
		_, err := tx.ExecContext(
			ctx,
			"INSERT INTO `sessions` (`id`, `code`, `delay_millis`, `body`, `created_at_millis`, `expires_at_millis`) VALUES (?, ?, ?, ?, ?, ?)", //nolint:lll
			sID,
			session.Code,
			session.Delay.Milliseconds(),
			session.ResponseBody,
			time.Now().UnixMilli(),
			time.Now().Add(s.sessionTTL).UnixMilli(),
		)
		if err != nil {
			return "", err
		}
	}

	{ // store headers using insert batch
		var args, values = make([]any, 0, len(session.Headers)*3), make([]string, 0, len(session.Headers)*3) //nolint:mnd
		for _, h := range session.Headers {
			args, values = append(args, sID, h.Name, h.Value), append(values, "(?, ?, ?)")
		}

		if len(args) > 0 {
			if _, err := tx.ExecContext(
				ctx,
				"INSERT INTO `response_headers` (`session_id`, `name`, `value`) VALUES "+strings.Join(values, ", "), //nolint:gosec
				args...,
			); err != nil {
				return "", err
			}
		}
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return "", commitErr
	}

	return sID, nil
}

func (s *SQLite) GetSession(ctx context.Context, sID string) (*Session, error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return nil, err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return nil, txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	var (
		session         Session
		expiresAtMillis int64
		delay           int64
	)

	if err := tx.QueryRowContext(
		ctx,
		"SELECT `code`, `delay_millis`, `body`, `created_at_millis`, `expires_at_millis` FROM `sessions` WHERE `id` = ?",
		sID,
	).Scan(&session.Code, &delay, &session.ResponseBody, &session.CreatedAtUnixMilli, &expiresAtMillis); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	session.Delay, session.ExpiresAt = time.Millisecond*time.Duration(delay), time.UnixMilli(expiresAtMillis)

	if session.ExpiresAt.Before(time.Now()) {
		if _, err := tx.ExecContext(ctx, "DELETE FROM `sessions` WHERE `id` = ?", sID); err != nil {
			return nil, err
		}

		if commitErr := commit(true); commitErr != nil { // commit the transaction
			return nil, commitErr
		}

		return nil, ErrSessionNotFound // session has been expired
	}

	// load session headers
	rows, qErr := tx.QueryContext(ctx, "SELECT `name`, `value` FROM `response_headers` WHERE `session_id` = ? ORDER BY `sequence` ASC", sID) //nolint:lll
	if qErr != nil {
		return nil, qErr
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h HttpHeader

		if sErr := rows.Scan(&h.Name, &h.Value); sErr != nil {
			return nil, sErr
		}

		session.Headers = append(session.Headers, h)
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return nil, commitErr
	}

	return &session, rows.Err()
}

func (s *SQLite) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	if _, err := tx.ExecContext(
		ctx,
		"UPDATE `sessions` SET `expires_at_millis` = (`expires_at_millis` + ?) WHERE `id` = ?",
		howMuch.Milliseconds(), sID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessionNotFound
		}

		return err
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return commitErr
	}

	return nil
}

func (s *SQLite) DeleteSession(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	if res, execErr := tx.ExecContext(ctx, "DELETE FROM `sessions` WHERE `id` = ?", sID); execErr != nil {
		if errors.Is(execErr, sql.ErrNoRows) {
			return ErrSessionNotFound
		}

		return execErr
	} else if affected, err := res.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return ErrSessionNotFound
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return commitErr
	}

	return nil
}

func (s *SQLite) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return "", err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return "", txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
		return "", err
	} else if !exists {
		return "", ErrSessionNotFound
	}

	rID = s.newID() // generate a new ID

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO `requests` (`id`, `session_id`, `method`, `client_address`, `url`, `payload`, `created_at_millis`) VALUES (?, ?, ?, ?, ?, ?, ?)", //nolint:lll
		rID,
		sID,
		r.Method,
		r.ClientAddr,
		r.URL,
		r.Body,
		time.Now().UnixMilli(),
	); err != nil {
		return "", err
	}

	// store request headers using insert batch
	var args, values = make([]any, 0, len(r.Headers)*3), make([]string, 0, len(r.Headers)*3) //nolint:mnd
	for _, h := range r.Headers {
		args, values = append(args, rID, h.Name, h.Value), append(values, "(?, ?, ?)")
	}

	if len(args) > 0 {
		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO `request_headers` (`request_id`, `name`, `value`) VALUES "+strings.Join(values, ", "), //nolint:gosec
			args...,
		); err != nil {
			return "", err
		}
	}

	{ // limit stored requests count
		var count int

		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(`id`) FROM `requests` WHERE `session_id` = ?",
			sID,
		).Scan(&count); err != nil {
			return "", err
		}

		if count > int(s.maxRequests) {
			// delete all requests from the requests table that are not in the last N requests, ordered by creation time
			if _, execErr := tx.ExecContext(
				ctx,
				"DELETE FROM `requests` WHERE `session_id` = ? AND `id` NOT IN (SELECT `id` FROM `requests` WHERE `session_id` = ? ORDER BY `sequence` DESC LIMIT ?)", //nolint:lll
				sID, sID, s.maxRequests,
			); execErr != nil {
				return "", execErr
			}
		}
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return "", commitErr
	}

	return rID, nil
}

func (s *SQLite) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return nil, err
	}

	var tx, commit, txErr = s.newTx(ctx, true)
	if txErr != nil {
		return nil, txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	// check the session existence
	if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	var request Request

	if err := tx.QueryRowContext(
		ctx,
		"SELECT `method`, `client_address`, `url`, `payload`, `created_at_millis` FROM `requests` WHERE `id` = ? AND `session_id` = ?", //nolint:lll
		rID, sID,
	).Scan(&request.Method, &request.ClientAddr, &request.URL, &request.Body, &request.CreatedAtUnixMilli); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}

		return nil, err
	}

	// load request headers
	rows, err := tx.QueryContext(ctx, "SELECT `name`, `value` FROM `request_headers` WHERE `request_id` = ? ORDER BY `sequence` ASC", rID) //nolint:lll
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

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return nil, commitErr
	}

	return &request, rows.Err()
}

func (s *SQLite) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return nil, err
	}

	var tx, commit, txErr = s.newTx(ctx, true)
	if txErr != nil {
		return nil, txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	// check the session existence
	if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	// read all stored request IDs
	rRows, rErr := tx.QueryContext(
		ctx,
		"SELECT `id`, `method`, `client_address`, `url`, `payload`, `created_at_millis` FROM `requests` WHERE `session_id` = ?", //nolint:lll
		sID,
	)
	if rErr != nil {
		return nil, rErr
	}

	defer func() { _ = rRows.Close() }()

	var all = make(map[string]Request)

	for rRows.Next() {
		var (
			request Request
			rID     string
		)

		if sErr := rRows.Scan(
			&rID,
			&request.Method,
			&request.ClientAddr,
			&request.URL,
			&request.Body,
			&request.CreatedAtUnixMilli,
		); sErr != nil {
			return nil, sErr
		}

		all[rID] = request
	}

	if err := rRows.Err(); err != nil {
		return nil, err
	}

	_ = rRows.Close() // close the rows asap

	// load all the request headers in a single query
	hRows, hErr := tx.QueryContext(
		ctx,
		"SELECT `request_id`, `name`, `value` FROM `request_headers` WHERE `request_id` IN (SELECT `id` FROM `requests` WHERE `session_id` = ?) ORDER BY `sequence` ASC", //nolint:lll
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

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return nil, commitErr
	}

	return all, hRows.Err()
}

func (s *SQLite) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	// check the session existence first
	if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	if res, execErr := tx.ExecContext(ctx, "DELETE FROM `requests` WHERE `id` = ? AND `session_id` = ?", rID, sID); execErr != nil { //nolint:lll
		if errors.Is(execErr, sql.ErrNoRows) {
			return ErrRequestNotFound
		}

		return execErr
	} else if affected, err := res.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return ErrRequestNotFound
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return commitErr
	}

	return nil
}

func (s *SQLite) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx, s); err != nil {
		return err
	}

	var tx, commit, txErr = s.newTx(ctx, false)
	if txErr != nil {
		return txErr
	}

	defer func() { _ = commit(false) }() // rollback the transaction in case of an error

	// check the session existence
	if exists, err := s.isSessionExists(ctx, tx, sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// delete all requests
	if _, err := tx.ExecContext(ctx, "DELETE FROM `requests` WHERE `session_id` = ?", sID); err != nil {
		return err
	}

	if commitErr := commit(true); commitErr != nil { // commit the transaction
		return commitErr
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

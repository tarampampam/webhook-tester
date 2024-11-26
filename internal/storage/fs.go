package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v2/internal/encoding"
)

type FS struct {
	sessionTTL      time.Duration
	maxRequests     uint32
	root            string
	cleanupInterval time.Duration
	encDec          encoding.EncoderDecoder

	// this function returns the current time, it's used to mock the time in tests
	timeNow func() time.Time

	close  chan struct{}
	closed atomic.Bool
}

const (
	fsDirPerm  = os.FileMode(0755)
	fsFilePerm = os.FileMode(0644)
)

var ( // ensure interface implementation
	_ Storage   = (*FS)(nil)
	_ io.Closer = (*FS)(nil)
)

type FSOption func(*FS)

func WithFSCleanupInterval(v time.Duration) FSOption { return func(f *FS) { f.cleanupInterval = v } }
func WithFSTimeNow(fn func() time.Time) FSOption     { return func(f *FS) { f.timeNow = fn } }

func NewFS(root string, sessionTTL time.Duration, maxRequests uint32, opts ...FSOption) *FS {
	var s = FS{
		root:            root,
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		cleanupInterval: time.Second, // default cleanup interval
		encDec:          encoding.JSON{},

		timeNow: func() time.Time { return time.Now().Round(time.Millisecond) }, // default time function, rounds to millis

		close: make(chan struct{}),
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
func (*FS) newID() string { return uuid.New().String() }

func (*FS) cleanup() {}

// isOpenAndNotDone checks if the storage is open and the context is not done.
func (s *FS) isOpenAndNotDone(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	} else if s.closed.Load() {
		return ErrClosed // storage is closed
	}

	return nil
}

// sessionDir returns the path to the session directory (e.g. {root}/{sID}).
func (s *FS) sessionDir(sID string) string { return path.Join(s.root, sID) }

// sessionFile returns the path to the session file (e.g. {root}/{sID}/session.json).
func (s *FS) sessionFile(sID string) string { return path.Join(s.sessionDir(sID), "session.json") }

// requestFile returns the path to the request file (e.g. {root}/{sID}/request-{rID}.json).
func (s *FS) requestFile(sID, rID string) string {
	return path.Join(s.sessionDir(sID), fmt.Sprintf("request-%s.json", rID))
}

func (s *FS) isSessionExists(sID string) (bool, error) {
	if stat, err := os.Stat(s.sessionFile(sID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("failed to stat session file: %w", err)
	} else if stat.IsDir() {
		return false, errors.New("session file is a directory")
	}

	return true, nil
}

func (s *FS) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err
	}

	if len(id) > 0 { // use the specified ID
		if len(id[0]) == 0 {
			return "", errors.New("empty session ID")
		}

		sID = id[0]

		if exists, err := s.isSessionExists(sID); err != nil {
			return "", err
		} else if exists {
			return "", errors.New("session already exists")
		}
	} else {
		sID = s.newID() // generate a new ID
	}

	var now = s.timeNow()

	// set the creation time
	session.CreatedAtUnixMilli = now.UnixMilli()

	// create a session directory
	if err := os.Mkdir(s.sessionDir(sID), fsDirPerm); err != nil {
		return "", err
	}

	// create a session file
	f, fErr := os.OpenFile(s.sessionFile(sID), os.O_WRONLY|os.O_CREATE, fsFilePerm)
	if fErr != nil {
		return "", fErr
	}

	defer func() { _ = f.Close() }()

	// encode the session data
	if data, mErr := s.encDec.Encode(session); mErr != nil {
		return "", mErr
	} else {
		// write the session data to the file
		if _, wErr := f.Write(data); wErr != nil {
			return "", wErr
		}
	}

	// close the file
	if err := f.Close(); err != nil {
		return "", err
	}

	// create a file with the TTL in unix milliseconds in its name
	if err := s.createTTLFile(s.sessionDir(sID), now.Add(s.sessionTTL)); err != nil {
		return "", err
	}

	return sID, nil
}

func (s *FS) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	var now = s.timeNow()

	// open the session file
	var f, fErr = os.OpenFile(s.sessionFile(sID), os.O_RDONLY, 0)
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fErr
	}

	defer func() { _ = f.Close() }()

	var ttl, tErr = s.getTTLFile(s.sessionDir(sID))
	if tErr != nil {
		return nil, tErr
	}

	// check the session expiration
	if ttl.Before(now) {
		if err := os.RemoveAll(s.sessionDir(sID)); err != nil {
			return nil, err
		}

		return nil, ErrSessionNotFound // session has been expired
	}

	// read the session data
	var data, rErr = io.ReadAll(f)
	if rErr != nil {
		return nil, rErr
	}

	// close the file
	if err := f.Close(); err != nil {
		return nil, err
	}

	// decode it
	var session Session
	if uErr := s.encDec.Decode(data, &session); uErr != nil {
		return nil, uErr
	}

	// set the expiration time
	session.ExpiresAt = *ttl

	return &session, nil
}

func (s *FS) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// delete the old TTL file
	if err := s.deleteTTLFile(s.sessionDir(sID)); err != nil {
		return err
	}

	// create a new TTL file
	return s.createTTLFile(s.sessionDir(sID), s.timeNow().Add(howMuch))
}

func (s *FS) DeleteSession(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// delete the session directory
	return os.RemoveAll(s.sessionDir(sID))
}

func (s *FS) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) { //nolint:funlen,gocyclo
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return "", err
	} else if !exists {
		return "", ErrSessionNotFound
	}

	var now = s.timeNow()

	rID, r.CreatedAtUnixMilli = s.newID(), now.UnixMilli()

	data, mErr := s.encDec.Encode(r)
	if mErr != nil {
		return "", mErr
	}

	// create a request file
	f, fErr := os.OpenFile(s.requestFile(sID, rID), os.O_WRONLY|os.O_CREATE, fsFilePerm)
	if fErr != nil {
		return "", fErr
	}

	defer func() { _ = f.Close() }()

	// write the request data to the file
	if _, wErr := f.Write(data); wErr != nil {
		return "", wErr
	}

	// close the file
	if err := f.Close(); err != nil {
		return "", err
	}

	{ // limit stored requests count
		var files, rErr = os.ReadDir(s.sessionDir(sID))
		if rErr != nil {
			return "", rErr
		}

		var requestFiles = make([]os.DirEntry, 0, len(files))

		// filter out request files
		for _, file := range files {
			if file.IsDir() || !file.Type().IsRegular() {
				continue
			}

			if n := file.Name(); strings.HasPrefix(n, "request-") && strings.HasSuffix(n, ".json") {
				requestFiles = append(requestFiles, file)
			}
		}

		if len(requestFiles) > int(s.maxRequests) {
			// sort list of files by modification time (newest first)
			slices.SortFunc(requestFiles, func(a, b os.DirEntry) int {
				aInfo, aErr := a.Info()
				if aErr != nil {
					return 0
				}

				bInfo, bErr := b.Info()
				if bErr != nil {
					return 0
				}

				return int(bInfo.ModTime().UnixMilli() - aInfo.ModTime().UnixMilli())
			})

			// remove unnecessary files
			for _, file := range requestFiles[s.maxRequests:] {
				if err := os.Remove(path.Join(s.sessionDir(sID), file.Name())); err != nil {
					return "", err
				}
			}
		}
	}

	return rID, nil
}

func (s *FS) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	// open the request file
	var f, fErr = os.OpenFile(s.requestFile(sID, rID), os.O_RDONLY, 0)
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrRequestNotFound
		}

		return nil, fErr
	}

	defer func() { _ = f.Close() }()

	// read the request data
	var data, rErr = io.ReadAll(f)
	if rErr != nil {
		return nil, rErr
	}

	// close the file
	if err := f.Close(); err != nil {
		return nil, err
	}

	// decode it
	var request Request
	if uErr := s.encDec.Decode(data, &request); uErr != nil {
		return nil, uErr
	}

	return &request, nil
}

func (s *FS) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	// read all stored request IDs
	var files, drErr = os.ReadDir(s.sessionDir(sID))
	if drErr != nil {
		return nil, drErr
	}

	var requests = make(map[string]Request, len(files))

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if n := file.Name(); strings.HasPrefix(n, "request-") && strings.HasSuffix(n, ".json") {
			var rID = strings.TrimSuffix(strings.TrimPrefix(n, "request-"), ".json")

			var f, fErr = os.OpenFile(path.Join(s.sessionDir(sID), n), os.O_RDONLY, 0)
			if fErr != nil {
				return nil, fErr
			}

			// read the request data
			var data, rErr = io.ReadAll(f)
			if rErr != nil {
				return nil, rErr
			}

			// close the file
			if err := f.Close(); err != nil {
				return nil, err
			}

			// decode it
			var request Request
			if uErr := s.encDec.Decode(data, &request); uErr != nil {
				return nil, uErr
			}

			requests[rID] = request
		}
	}

	return requests, nil
}

func (s *FS) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// delete the request file
	if err := os.Remove(s.requestFile(sID, rID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrRequestNotFound
		}

		return err
	}

	return nil
}

func (s *FS) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return err
	} else if !exists {
		return ErrSessionNotFound
	}

	// read all stored request IDs
	var files, drErr = os.ReadDir(s.sessionDir(sID))
	if drErr != nil {
		return drErr
	}

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if n := file.Name(); strings.HasPrefix(n, "request-") && strings.HasSuffix(n, ".json") {
			if err := os.Remove(path.Join(s.sessionDir(sID), n)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *FS) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		close(s.close)

		return nil
	}

	return ErrClosed
}

const fsTTLFileExt = ".ttl"

// createTTLFile creates a file in the directory with the TTL in its name.
func (*FS) createTTLFile(dir string, ttl time.Time) error {
	f, err := os.OpenFile(
		path.Join(dir, strconv.FormatInt(ttl.UnixMilli(), 10)+fsTTLFileExt),
		os.O_CREATE,
		fsFilePerm,
	)
	if err != nil {
		return fmt.Errorf("failed to create TTL file: %w", err)
	}

	return f.Close()
}

// getTTLFile returns the TTL, stored in the name of the file in the directory.
func (*FS) getTTLFile(dir string) (*time.Time, error) {
	// find a file with the TTL in its name
	var files, rErr = os.ReadDir(dir)
	if rErr != nil {
		return nil, rErr
	}

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if strings.HasSuffix(file.Name(), fsTTLFileExt) {
			var ttl, pErr = strconv.ParseInt(strings.TrimSuffix(file.Name(), fsTTLFileExt), 10, 64)
			if pErr != nil {
				return nil, fmt.Errorf("failed to parse TTL file name: %w", pErr)
			}

			var t = time.UnixMilli(ttl)

			return &t, nil
		}
	}

	return nil, errors.New("ttl file not found")
}

// deleteTTLFile finds and deletes the TTL file in the directory. If the file is not found, it does nothing.
func (*FS) deleteTTLFile(dir string) error {
	// find a file with the TTL in its name
	var files, rErr = os.ReadDir(dir)
	if rErr != nil {
		return fmt.Errorf("failed to read directory: %w", rErr)
	}

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if strings.HasSuffix(file.Name(), fsTTLFileExt) {
			if err := os.Remove(path.Join(dir, file.Name())); err != nil {
				return fmt.Errorf("failed to remove TTL file: %w", err)
			}
		}
	}

	return nil
}

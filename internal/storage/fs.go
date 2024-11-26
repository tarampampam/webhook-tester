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
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v2/internal/encoding"
)

// FS is an implementation of the Storage interface that organizes data on the filesystem using the following structure:
//
//	📂 {root}
//	├── 📂 {session-uuid}
//	│   ├── 📄 session.<expiration-time-unix-millis>.json
//	│   ├── 📄 request.<created-time-unix-millis>.{request-uuid}.json
//	│   └── …
//	└── …
type FS struct {
	sessionTTL      time.Duration
	maxRequests     uint32
	root            string
	cleanupInterval time.Duration
	encDec          encoding.EncoderDecoder
	sessionsMu      syncMap[string /* sID */, *sync.RWMutex] // used to protect the session data (expiration time at least)

	// this function returns the current time, it's used to mock the time in tests
	timeNow func() time.Time

	close  chan struct{}
	closed atomic.Bool

	dirPerm, filePerm os.FileMode
}

var ( // ensure interface implementation
	_ Storage   = (*FS)(nil)
	_ io.Closer = (*FS)(nil)
)

type FSOption func(*FS)

func WithFSCleanupInterval(v time.Duration) FSOption { return func(f *FS) { f.cleanupInterval = v } }
func WithFSTimeNow(fn func() time.Time) FSOption     { return func(f *FS) { f.timeNow = fn } }
func WithFSDirPerm(v os.FileMode) FSOption           { return func(f *FS) { f.dirPerm = v } }
func WithFSFilePerm(v os.FileMode) FSOption          { return func(f *FS) { f.filePerm = v } }

func NewFS(root string, sessionTTL time.Duration, maxRequests uint32, opts ...FSOption) *FS {
	var s = FS{
		root:            root,
		sessionTTL:      sessionTTL,
		maxRequests:     maxRequests,
		cleanupInterval: time.Second, // default cleanup interval
		encDec:          encoding.JSON{},
		timeNow:         defaultTimeFunc,
		close:           make(chan struct{}),
		dirPerm:         os.FileMode(0755), //nolint:mnd
		filePerm:        os.FileMode(0644), //nolint:mnd
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

func (*FS) cleanup() {} // TODO: implement the cleanup function

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

func (s *FS) isSessionExists(sID string) (bool, error) {
	stat, err := os.Stat(s.sessionDir(sID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return stat.Mode().IsDir(), nil
}

// findSessionFile returns the path to the session file with the expiration time and the expiration time itself.
// If the session file is not found, the function returns os.ErrNotExist.
func (s *FS) findSessionFile(sID string) (filePath string, _ *time.Time, _ error) {
	var dir = s.sessionDir(sID)

	files, rErr := os.ReadDir(dir)
	if rErr != nil {
		return "", nil, rErr
	}

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if n := file.Name(); strings.HasPrefix(n, "session.") && strings.HasSuffix(n, ".json") {
			var ts, err = strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(n, "session."), ".json"), 10, 64)
			if err != nil {
				return "", nil, err
			}

			var t = time.UnixMilli(ts)

			return path.Join(dir, n), &t, nil
		}
	}

	return "", nil, os.ErrNotExist
}

type fsRequestFile struct {
	rID, path string
	createdAt time.Time
}

// listRequestFiles returns a list of request files for the specified session ID. The list is sorted by creation time
// (newest first).
func (s *FS) listRequestFiles(sID string) ([]fsRequestFile, error) {
	var dir = s.sessionDir(sID)

	var files, rErr = os.ReadDir(dir)
	if rErr != nil {
		return nil, rErr
	}

	var list = make([]fsRequestFile, 0, len(files))

	// filter out request files
	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		// file format: request.<created-time-unix-millis>.{request-uuid}.json
		if n := file.Name(); strings.HasPrefix(n, "request.") && strings.HasSuffix(n, ".json") {
			var parts = strings.Split(strings.TrimSuffix(strings.TrimPrefix(n, "request."), ".json"), ".")
			if len(parts) != 2 { //nolint:mnd
				continue
			}

			var ts, tsErr = strconv.ParseInt(parts[0], 10, 64)
			if tsErr != nil {
				continue
			}

			list = append(list, fsRequestFile{
				rID:       parts[1],
				path:      path.Join(dir, n),
				createdAt: time.UnixMilli(ts),
			})
		}
	}

	// sort the list by creation time (newest first)
	slices.SortFunc(list, func(a, b fsRequestFile) int { return int(b.createdAt.UnixMilli() - a.createdAt.UnixMilli()) })

	return list, nil
}

// lockSession lazy initializes a mutex for the session (if needed) and locks it, regarding the readOnly flag. The
// returned function should be called to unlock the mutex.
func (s *FS) lockSession(sID string, readOnly bool) func() {
	var mu *sync.RWMutex

	if v, ok := s.sessionsMu.Load(sID); ok { // lazy initialization
		mu = v
	} else {
		mu = new(sync.RWMutex)
		s.sessionsMu.Store(sID, mu)
	}

	if readOnly {
		mu.RLock()
	} else {
		mu.Lock()
	}

	return func() {
		if readOnly {
			mu.RUnlock()
		} else {
			mu.Unlock()
		}
	}
}

func (s *FS) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) { //nolint:funlen
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

	var (
		now        = s.timeNow()
		sessionDir = s.sessionDir(sID)
	)

	// set the creation time
	session.CreatedAtUnixMilli = now.UnixMilli()

	// create a session directory
	if err := os.Mkdir(sessionDir, s.dirPerm); err != nil {
		return "", err
	}

	// create a session file
	f, fErr := os.OpenFile(
		path.Join(sessionDir, fmt.Sprintf("session.%d.json", now.Add(s.sessionTTL).UnixMilli())),
		os.O_WRONLY|os.O_CREATE,
		s.filePerm,
	)
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

	// for new sessions, create a mutex without checking if it already exists because it doesn't
	s.sessionsMu.Store(sID, new(sync.RWMutex))

	return sID, nil
}

func (s *FS) GetSession(ctx context.Context, sID string) (*Session, error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrSessionNotFound
	}

	defer s.lockSession(sID, false)()

	// find the session file
	filePath, expiresAt, sErr := s.findSessionFile(sID)
	if sErr != nil {
		if errors.Is(sErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, sErr
	}

	// check the session expiration
	if expiresAt.Before(s.timeNow()) {
		if err := os.RemoveAll(s.sessionDir(sID)); err != nil {
			return nil, err
		}

		return nil, ErrSessionNotFound // session has been expired
	}

	// open the session file
	file, fErr := os.OpenFile(filePath, os.O_RDONLY, 0)
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fErr
	}

	defer func() { _ = file.Close() }()

	// read the content
	data, rErr := io.ReadAll(file)
	if rErr != nil {
		return nil, rErr
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	// decode
	var session Session
	if uErr := s.encDec.Decode(data, &session); uErr != nil {
		return nil, uErr
	}

	// set the expiration time
	session.ExpiresAt = *expiresAt

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

	defer s.lockSession(sID, false)()

	// find the session file
	filePath, _, sErr := s.findSessionFile(sID)
	if sErr != nil {
		if errors.Is(sErr, os.ErrNotExist) {
			return ErrSessionNotFound
		}

		return sErr
	}

	// rename the session file, to store the new expiration time
	return os.Rename(
		filePath,
		path.Join(path.Dir(filePath), fmt.Sprintf("session.%d.json", s.timeNow().Add(howMuch).UnixMilli())),
	)
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

	unlock := s.lockSession(sID, false)

	defer func() { unlock(); s.sessionsMu.Delete(sID) }()

	// delete the session directory
	return os.RemoveAll(s.sessionDir(sID))
}

func (s *FS) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err
	}

	if exists, err := s.isSessionExists(sID); err != nil {
		return "", err
	} else if !exists {
		return "", ErrSessionNotFound
	}

	defer s.lockSession(sID, false)()

	rID, r.CreatedAtUnixMilli = s.newID(), s.timeNow().UnixMilli()

	data, mErr := s.encDec.Encode(r)
	if mErr != nil {
		return "", mErr
	}

	var dir = s.sessionDir(sID)

	// create a request file
	f, fErr := os.OpenFile(
		path.Join(dir, fmt.Sprintf("request.%d.%s.json", r.CreatedAtUnixMilli, rID)),
		os.O_WRONLY|os.O_CREATE,
		s.filePerm,
	)
	if fErr != nil {
		return "", fErr
	}

	defer func() { _ = f.Close() }()

	// write the request data to the file
	if _, wErr := f.Write(data); wErr != nil {
		return "", wErr
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	{ // limit stored requests count
		list, lErr := s.listRequestFiles(sID)
		if lErr != nil {
			return "", lErr
		}

		if len(list) > int(s.maxRequests) {
			// remove unnecessary files
			for _, file := range list[s.maxRequests:] {
				if err := os.Remove(file.path); err != nil {
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

	defer s.lockSession(sID, true)()

	var dir = s.sessionDir(sID)

	// find the request file
	var files, rdErr = os.ReadDir(dir)
	if rdErr != nil {
		return nil, rdErr
	}

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue
		}

		if n := file.Name(); strings.HasPrefix(n, "request.") && strings.Contains(n, rID) {
			f, fErr := os.OpenFile(path.Join(dir, n), os.O_RDONLY, 0)
			if fErr != nil {
				return nil, fErr
			}

			// read the request data
			data, rErr := io.ReadAll(f)
			if rErr != nil {
				_ = f.Close() // do not forget to close the file in case of an error

				return nil, rErr
			}

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
	}

	return nil, ErrRequestNotFound
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

	defer s.lockSession(sID, true)()

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return nil, lErr
	}

	var requests = make(map[string]Request, len(list))

	for _, file := range list {
		f, fErr := os.OpenFile(file.path, os.O_RDONLY, 0)
		if fErr != nil {
			return nil, fErr
		}

		// read the request data
		var data, rErr = io.ReadAll(f)
		if rErr != nil {
			_ = f.Close() // do not forget to close the file in case of an error

			return nil, rErr
		}

		if err := f.Close(); err != nil {
			return nil, err
		}

		// decode it
		var request Request
		if uErr := s.encDec.Decode(data, &request); uErr != nil {
			return nil, uErr
		}

		requests[file.rID] = request
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

	defer s.lockSession(sID, false)()

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return lErr
	}

	for _, file := range list {
		if file.rID == rID {
			if err := os.Remove(file.path); err != nil {
				return err
			}

			return nil
		}
	}

	return ErrRequestNotFound
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

	defer s.lockSession(sID, false)()

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return lErr
	}

	for _, file := range list {
		if err := os.Remove(file.path); err != nil {
			return err
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

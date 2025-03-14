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
	"golang.org/x/sync/errgroup"

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
	mu              sync.RWMutex

	// this function returns the current time, it's used to mock the time in tests
	timeNow TimeFunc

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
func WithFSTimeNow(fn TimeFunc) FSOption             { return func(f *FS) { f.timeNow = fn } }

//	func WithFSDirPerm(v os.FileMode) FSOption       { return func(f *FS) { f.dirPerm = v } }
//	func WithFSFilePerm(v os.FileMode) FSOption      { return func(f *FS) { f.filePerm = v } }

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
		go s.cleanup(context.Background()) // start cleanup goroutine
	}

	return &s
}

// newID generates a new (unique) ID.
func (*FS) newID() string { return uuid.New().String() }

// withLock lock the mutex for reading or writing and calls the specified function.
// The readOnly parameter specifies whether the lock is read-only.
//
// The function returns the result of the function call.
func (s *FS) withLock(readOnly bool, fn func() error) error {
	if readOnly {
		s.mu.RLock()
	} else {
		s.mu.Lock()
	}

	defer func() {
		if readOnly {
			s.mu.RUnlock()
		} else {
			s.mu.Unlock()
		}
	}()

	return fn()
}

func (s *FS) cleanup(ctx context.Context) {
	var timer = time.NewTimer(s.cleanupInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.close: // close signal received
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			var (
				now  = s.timeNow()
				dirs []os.DirEntry
				sIDs []string
			)

			// list all session directories
			if err := s.withLock(true, func() (err error) { dirs, err = os.ReadDir(s.root); return }); err == nil { //nolint:nlreturn,lll
				sIDs = make([]string, 0, len(dirs))

				for _, dir := range dirs {
					if dir.IsDir() && len(dir.Name()) == 36 { // UUID length
						sIDs = append(sIDs, dir.Name())
					}
				}
			}

			var wg sync.WaitGroup

			for _, sID := range sIDs {
				wg.Add(1)

				go func() {
					defer wg.Done()

					// check the session expiration
					if _, expiresAt, err := s.findSessionFile(sID); err == nil && expiresAt.Before(now) {
						_ = s.DeleteSession(ctx, sID) // and delete the expired
					}
				}()
			}

			wg.Wait()
			timer.Reset(s.cleanupInterval)
		}
	}
}

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

func (s *FS) findSessionFile(sID string) (filePath string, _ *time.Time, _ error) {
	var (
		dir   = s.sessionDir(sID)
		files []os.DirEntry
	)

	if err := s.withLock(true, func() (err error) { files, err = os.ReadDir(dir); return }); err != nil { //nolint:nlreturn,lll
		return "", nil, err // directory reading failed
	}

	const prefix, postfix = "session.", ".json"

	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue // is not a regular file
		}

		if n := file.Name(); strings.HasPrefix(n, prefix) && strings.HasSuffix(n, postfix) {
			var ts, err = strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(n, prefix), postfix), 10, 64)
			if err != nil {
				return "", nil, err // timestamp parsing failed
			}

			var t = time.UnixMilli(ts)

			return path.Join(dir, n), &t, nil
		}
	}

	return "", nil, os.ErrNotExist // no file found
}

type fsRequestFile struct {
	rID, path string
	createdAt time.Time
}

// listRequestFiles returns a list of request files for the specified session ID. The list is sorted by creation time
// (newest first).
func (s *FS) listRequestFiles(sID string) ([]fsRequestFile, error) {
	var (
		dir   = s.sessionDir(sID)
		files []os.DirEntry
	)

	if err := s.withLock(true, func() (err error) { files, err = os.ReadDir(dir); return }); err != nil { //nolint:nlreturn,lll
		return nil, err // directory reading failed
	}

	var list = make([]fsRequestFile, 0, len(files)-1) // -1 because we don't count the session file

	const prefix, postfix = "request.", ".json"

	// filter out request files
	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() {
			continue // is not a regular file
		}

		// file format: request.<created-time-unix-millis>.{request-uuid}.json
		if n := file.Name(); strings.HasPrefix(n, prefix) && strings.HasSuffix(n, postfix) {
			var parts = strings.Split(strings.TrimSuffix(strings.TrimPrefix(n, prefix), postfix), ".")
			if len(parts) != 2 { //nolint:mnd
				continue // invalid file name
			}

			var ts, tsErr = strconv.ParseInt(parts[0], 10, 64)
			if tsErr != nil {
				continue // timestamp parsing failed
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

func (s *FS) NewSession(ctx context.Context, session Session, id ...string) (sID string, _ error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err // closed, or context is done
	}

	var now = s.timeNow()

	if len(id) > 0 { //nolint:nestif // use the specified ID
		if len(id[0]) == 0 {
			return "", errors.New("empty session ID")
		}

		sID = id[0]

		if _, expiresAt, err := s.findSessionFile(sID); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", err // unexpected error (ignore "session not found" error)
			}
		} else if expiresAt.Before(now) { // session found, but expired
			if dErr := s.DeleteSession(ctx, sID); dErr != nil {
				return "", dErr
			}
		} else { // no error, not expired == session already exists
			return "", errors.New("session already exists")
		}
	} else {
		sID = s.newID() // generate a new ID
	}

	// set the creation time
	session.CreatedAtUnixMilli = now.UnixMilli()

	// encode the session data
	data, mErr := s.encDec.Encode(session)
	if mErr != nil {
		return "", mErr
	}

	if err := s.withLock(false, func() error {
		var sessionDir = s.sessionDir(sID)

		// create a session directory
		if err := os.Mkdir(sessionDir, s.dirPerm); err != nil {
			return err
		}

		// create a session file
		f, fErr := os.OpenFile(
			path.Join(sessionDir, fmt.Sprintf("session.%d.json", now.Add(s.sessionTTL).UnixMilli())),
			os.O_WRONLY|os.O_CREATE,
			s.filePerm,
		)
		if fErr != nil {
			return fErr
		}

		defer func() { _ = f.Close() }()

		// write the data to the file
		if _, err := f.Write(data); err != nil {
			return fErr
		}

		return f.Close()
	}); err != nil {
		return "", err
	}

	return sID, nil
}

func (s *FS) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err // closed, or context is done
	}

	var now = s.timeNow()

	filePath, expiresAt, sErr := s.findSessionFile(sID)
	if sErr != nil {
		if errors.Is(sErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, sErr
	} else if expiresAt.Before(now) { // check the session expiration
		if err := s.DeleteSession(ctx, sID); err != nil {
			return nil, err
		}

		return nil, ErrSessionNotFound // session has been expired
	}

	var data []byte

	if err := s.withLock(true, func() (err error) {
		var f *os.File

		if f, err = os.OpenFile(filePath, os.O_RDONLY, 0); err != nil {
			return // file opening failed
		}

		defer func() { _ = f.Close() }()

		if data, err = io.ReadAll(f); err != nil {
			return // failed to read the file
		}

		return f.Close()
	}); err != nil {
		if errors.Is(err, os.ErrNotExist) { // probably, another thread has deleted the session
			return nil, ErrSessionNotFound
		}

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
		return err // closed, or context is done
	}

	var now = s.timeNow()

	filePath, expiresAt, sErr := s.findSessionFile(sID)
	if sErr != nil {
		if errors.Is(sErr, os.ErrNotExist) {
			return ErrSessionNotFound
		}

		return sErr
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return dErr
		}

		return ErrSessionNotFound
	}

	// rename the session file, to store the new expiration time
	return s.withLock(false, func() error {
		return os.Rename(
			filePath,
			path.Join(path.Dir(filePath), fmt.Sprintf("session.%d.json", expiresAt.Add(howMuch).UnixMilli())),
		)
	})
}

func (s *FS) DeleteSession(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err // closed, or context is done
	}

	if _, _, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSessionNotFound
		}

		return err
	}

	// delete the session directory with all its content
	return s.withLock(false, func() error { return os.RemoveAll(s.sessionDir(sID)) })
}

func (s *FS) NewRequest(ctx context.Context, sID string, r Request) (rID string, _ error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return "", err
	}

	var now = s.timeNow()

	// check the session existence
	if _, expiresAt, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrSessionNotFound
		}

		return "", err
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return "", dErr
		}

		return "", ErrSessionNotFound
	}

	rID, r.CreatedAtUnixMilli = s.newID(), now.UnixMilli()

	data, mErr := s.encDec.Encode(r)
	if mErr != nil {
		return "", mErr
	}

	if err := s.withLock(false, func() (err error) {
		var (
			dir = s.sessionDir(sID)
			f   *os.File
		)

		// create a request file
		if f, err = os.OpenFile(
			path.Join(dir, fmt.Sprintf("request.%d.%s.json", r.CreatedAtUnixMilli, rID)),
			os.O_WRONLY|os.O_CREATE,
			s.filePerm,
		); err != nil {
			return
		}

		defer func() { _ = f.Close() }()

		// write the request data to the file
		if _, err = f.Write(data); err != nil {
			return
		}

		return f.Close()
	}); err != nil {
		return "", err
	}

	if s.maxRequests > 0 { // limit stored requests count
		list, lErr := s.listRequestFiles(sID)
		if lErr != nil {
			return "", lErr
		}

		if len(list) > int(s.maxRequests) {
			var toRemove = list[s.maxRequests:]

			// remove unnecessary files
			if err := s.withLock(false, func() (err error) {
				for _, file := range toRemove {
					if err = os.Remove(file.path); err != nil {
						return err // return the first error
					}
				}

				return
			}); err != nil {
				return "", err // failed to remove files
			}
		}
	}

	return rID, nil
}

func (s *FS) GetRequest(ctx context.Context, sID, rID string) (*Request, error) { //nolint:funlen,gocyclo,gocognit
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	var now = s.timeNow()

	// check the session existence
	if _, expiresAt, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return nil, dErr
		}

		return nil, ErrSessionNotFound
	}

	var data []byte

	if err := s.withLock(true, func() (err error) {
		var (
			dir   = s.sessionDir(sID)
			files []os.DirEntry
		)

		if files, err = os.ReadDir(dir); err != nil {
			return // directory reading failed
		}

		for _, file := range files {
			if file.IsDir() || !file.Type().IsRegular() {
				continue // is not a regular file
			}

			if n := file.Name(); strings.HasPrefix(n, "request.") && strings.Contains(n, rID) {
				var f *os.File

				if f, err = os.OpenFile(path.Join(dir, n), os.O_RDONLY, 0); err != nil {
					return // file opening failed
				}

				if d, rErr := io.ReadAll(f); rErr != nil {
					_ = f.Close() // do not forget to close the file in case of an error

					return rErr // reading failed
				} else {
					data = d
				}

				return f.Close()
			}
		}

		return
	}); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, ErrRequestNotFound // request not found (no data)
	}

	var request Request
	if uErr := s.encDec.Decode(data, &request); uErr != nil {
		return nil, uErr
	}

	return &request, nil
}

func (s *FS) GetAllRequests(ctx context.Context, sID string) (map[string]Request, error) { //nolint:funlen
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return nil, err
	}

	var now = s.timeNow()

	// check the session existence
	if _, expiresAt, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return nil, dErr
		}

		return nil, ErrSessionNotFound
	}

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return nil, lErr
	}

	var (
		eg errgroup.Group
		mu sync.Mutex // protect the map
		m  = make(map[string]Request, len(list))
	)

	for _, file := range list {
		eg.Go(func() error {
			var data []byte

			if err := s.withLock(true, func() (err error) {
				var f *os.File

				if f, err = os.OpenFile(file.path, os.O_RDONLY, 0); err != nil {
					return // file opening failed
				}

				if data, err = io.ReadAll(f); err != nil {
					_ = f.Close() // do not forget to close the file in case of an error

					return // reading failed
				}

				return f.Close()
			}); err != nil {
				return err
			}

			var request Request
			if err := s.encDec.Decode(data, &request); err != nil {
				return err // decoding failed
			}

			mu.Lock()
			m[file.rID] = request
			mu.Unlock()

			return nil
		})
	}

	return m, eg.Wait()
}

func (s *FS) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	var now = s.timeNow()

	// check the session existence
	if _, expiresAt, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSessionNotFound
		}

		return err
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return dErr
		}

		return ErrSessionNotFound
	}

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return lErr
	}

	for _, file := range list {
		if file.rID == rID {
			return s.withLock(false, func() error { return os.Remove(file.path) })
		}
	}

	return ErrRequestNotFound
}

func (s *FS) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.isOpenAndNotDone(ctx); err != nil {
		return err
	}

	var now = s.timeNow()

	// check the session existence
	if _, expiresAt, err := s.findSessionFile(sID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSessionNotFound
		}

		return err
	} else if expiresAt.Before(now) {
		if dErr := s.DeleteSession(ctx, sID); dErr != nil { // delete the expired session
			return dErr
		}

		return ErrSessionNotFound
	}

	// list all request files
	var list, lErr = s.listRequestFiles(sID)
	if lErr != nil {
		return lErr
	}

	if err := s.withLock(false, func() (err error) {
		for _, file := range list {
			if err = os.Remove(file.path); err != nil {
				return // return the first error
			}
		}

		return
	}); err != nil {
		return err // failed to remove files
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

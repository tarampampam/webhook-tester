package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5" //nolint:gosec // MD5 is used for non-cryptographic hashing of filesystem paths
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	fsDirPerm  = os.FileMode(0o755) // -rwxr-xr-x
	fsFilePerm = os.FileMode(0o644) // -rw-r--r--

	readDirSize            = 128 // number of entries to read per ReadDir call when scanning directories
	requestMetaFilePostfix = ".meta.bin"
)

// FS is a filesystem-backed [Storage] implementation that organizes data on the filesystem using the following layout:
//
//	📂 {root}
//	├── 📂 {md5(sID)}                   // 32-char lowercase hex directory per session
//	│   ├── 📄 meta.bin                 // binary session metadata
//	│   ├── 📄 session.json[.gz]        // session response payload
//	│   └── 📂 requests
//	│       ├── 📄 index.txt            // request index, one line per request
//	│       ├── 📄 {md5(rID)}.meta.bin  // binary request metadata
//	│       └── 📄 {md5(rID)}.json[.gz] // captured request payload
//	└── …
//
// A background goroutine evicts expired sessions on a configurable interval; it stops when [FS.Close] is called or
// the context passed to [NewFS] is canceled. After either event all methods return [ErrClosed].
type FS struct {
	root          *os.Root
	requestsLimit uint
	compressor    Compressor
	decBuf        *pool[*bytes.Buffer]
	timeNow       func() time.Time
	cleanupTick   time.Duration
	mu            sync.RWMutex // TODO: make mutex file-based to allow multiple FS instances to share the same root

	closeOnce     sync.Once     // guards closeSignalCh from being closed multiple times
	closeSignalCh chan struct{} // closed to signal the cleanup goroutine to stop
	closedCh      chan struct{} // closed when the cleanup goroutine has stopped, to allow Close to wait for it
}

var ( // compile-time interface assertion
	_ Storage   = (*FS)(nil)
	_ io.Closer = (*FS)(nil)
)

// FSOption configures an [FS] storage instance.
type FSOption func(*FS)

// WithFSTimeNow sets the function that returns the current time.
func WithFSTimeNow(fn func() time.Time) FSOption { return func(s *FS) { s.timeNow = fn } }

// WithFSCleanupInterval sets the interval between expired-session cleanup sweeps.
// Defaults to 1 second.
func WithFSCleanupInterval(d time.Duration) FSOption { return func(s *FS) { s.cleanupTick = d } }

// WithFSCompressor sets the compressor used for session and request data files.
// Defaults to [GzipCompressor].
func WithFSCompressor(c Compressor) FSOption { return func(s *FS) { s.compressor = c } }

// NewFS creates a new filesystem-backed storage instance rooted at root.
// The caller retains ownership of root; [FS.Close] does not close it.
//
// `requestsLimit` is the maximum number of captured requests stored per session (0 = unlimited).
// The cleanup goroutine stops when Close is called or ctx is canceled; either event closes the storage.
func NewFS(ctx context.Context, root *os.Root, requestsLimit uint, opts ...FSOption) *FS { //nolint:gocognit
	s := &FS{
		root:          root,
		requestsLimit: requestsLimit,
		compressor:    NewGzipCompressor(gzip.BestSpeed),
		decBuf:        newPool[*bytes.Buffer](func() *bytes.Buffer { return new(bytes.Buffer) }),
		timeNow:       time.Now,
		cleanupTick:   time.Second,
		closeSignalCh: make(chan struct{}),
		closedCh:      make(chan struct{}),
	}

	for _, o := range opts {
		o(s)
	}

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
				s.mu.Lock()

				now := s.timeNow()

				if rootDir, oErr := s.root.Open("."); oErr == nil {
					for {
						dirs, rdErr := rootDir.ReadDir(readDirSize)
						if rdErr != nil && !errors.Is(rdErr, io.EOF) {
							break // something went wrong reading the root directory
						}

						for _, d := range dirs {
							if !d.IsDir() || !s.isValidHash(d.Name()) {
								continue
							}

							if meta, mErr := s.readSessionMetaH(d.Name()); mErr == nil && meta.IsExpired(now) {
								_ = s.root.RemoveAll(d.Name()) //nolint:errcheck
							}
						}

						if rdErr != nil {
							break // includes io.EOF
						}
					}

					_ = rootDir.Close()
				}

				s.mu.Unlock()
			}
		}
	}()

	return s
}

// Reindex rebuilds all per-session request indexes by scanning the filesystem.
//
// It is safe to call on a clean (empty) directory - no index files are written if no session directories are found.
//
// Callers should invoke Reindex once after [NewFS] when crash recovery is desired - a clean shutdown keeps request
// indexes consistent, but an unexpected process termination may leave them stale. Reindex is not required for a
// first-time startup on an empty directory, and is not needed on normal (non-crash) restarts.
//
// Reindex holds the write lock for its entire duration, blocking all concurrent reads and writes.
//
// If ctx is canceled mid-scan, request indexes for sessions already processed will have been updated.
// Reindex always returns a non-nil error in that case; callers that require a consistent index must call it again.
func (s *FS) Reindex(ctx context.Context) error { //nolint:gocognit,funlen
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	const ioWorkers = 32 // optimal queue depth for NVMe

	// semaphore limits concurrent request-reindex goroutines
	sem := make(chan struct{}, ioWorkers)
	defer close(sem)

	var (
		outErr atomic.Pointer[error]
		wg     sync.WaitGroup
	)

	root, oErr := s.root.Open(".")
	if oErr != nil {
		return fmt.Errorf("open root directory: %w", oErr)
	}

	defer func() { _ = root.Close() }()

rootDirsLoop:
	for { // iterate over all files and directories in the root
		if err := ctx.Err(); err != nil {
			outErr.Store(&err)

			break
		}

		rootDirs, rootDirsErr := root.ReadDir(readDirSize)
		if rootDirsErr != nil && !errors.Is(rootDirsErr, io.EOF) {
			outErr.Store(&rootDirsErr)

			break
		}

		for _, rootDir := range rootDirs {
			if err := ctx.Err(); err != nil {
				outErr.Store(&err)

				break rootDirsLoop
			}

			// skip non-directories, and directories that don't look like session dirs
			if !rootDir.IsDir() || !s.isValidHash(rootDir.Name()) {
				continue
			}

			// read the session metadata file to skip expired sessions
			sm, smErr := s.root.Open(s.sessionMetaFileH(rootDir.Name()))
			if smErr != nil {
				if errors.Is(smErr, os.ErrNotExist) {
					continue // skip partially created or already removed
				}

				outErr.Store(new(fmt.Errorf("open session metadata %s: %w", rootDir.Name(), smErr)))

				break rootDirsLoop
			}

			var sessionMeta fsSessionMeta

			_, mrErr := sessionMeta.ReadFrom(sm)
			_ = sm.Close()

			if mrErr != nil || sessionMeta.IsExpired(s.timeNow()) {
				continue // skip expired sessions and sessions with unreadable metadata file
			}

			sem <- struct{}{} // acquire a goroutine slot

			wg.Add(1)

			// reindex the requests for this session in a separate goroutine to speed up the process
			go func(sHash string) {
				defer func() { <-sem; wg.Done() }() // release the goroutine slot when done

				if ctx.Err() != nil { // respect context cancellation
					return
				}

				reqDir, reqDirErr := s.root.Open(s.requestsDirH(sHash))
				if reqDirErr != nil {
					if !errors.Is(reqDirErr, os.ErrNotExist) {
						outErr.Store(new(fmt.Errorf("open requests dir %s: %w", sHash, reqDirErr)))
					}

					return // no requests directory - nothing to index
				}

				defer func() { _ = reqDir.Close() }()

				if err := newFSRequestIndex(s.root, s.requestsDirH(sHash)).WriteIndex(func(yield func(fsRequestIndexRecord) bool) {
					for {
						if ctx.Err() != nil {
							return
						}

						// read requests directories
						reqFiles, rdErr := reqDir.ReadDir(readDirSize)
						if rdErr != nil && !errors.Is(rdErr, io.EOF) {
							outErr.Store(new(fmt.Errorf("read requests dir %s: %w", sHash, rdErr)))

							return
						}

						for _, reqFile := range reqFiles {
							rHash, ok := s.requestHashFromFilename(reqFile.Name())
							if !reqFile.Type().IsRegular() || !ok {
								continue
							}

							// read request metadata file
							rmf, rmfErr := s.root.Open(s.requestMetaFileH(sHash, rHash))
							if rmfErr != nil {
								if errors.Is(rmfErr, os.ErrNotExist) {
									continue
								}

								outErr.Store(new(fmt.Errorf("open request metadata %s/%s: %w", sHash, rHash, rmfErr)))

								return
							}

							var requestMeta fsRequestMeta

							_, readErr := requestMeta.ReadFrom(rmf)
							_ = rmf.Close()

							if readErr != nil {
								continue // skip requests with unreadable metadata
							}

							// pass the request index entry to the index writer
							if !yield(fsRequestIndexRecord{Hash: rHash, CreatedAt: requestMeta.CreatedAt}) {
								return
							}
						}

						if errors.Is(rdErr, io.EOF) {
							return
						}
					}
				}); err != nil {
					outErr.Store(new(fmt.Errorf("write requests index for session %s: %w", sHash, err)))
				}
			}(rootDir.Name())
		}

		if errors.Is(rootDirsErr, io.EOF) {
			break // end of directory reached
		}
	}

	wg.Wait()

	if err := outErr.Load(); err != nil && *err != nil {
		return *err
	}

	return ctx.Err() // non-nil if context was canceled after the sessions scan but during goroutines
}

// checkOpen returns [ErrClosed] if the storage has been closed, or ctx.Err() if the context is done.
func (s *FS) checkOpen(ctx context.Context) error {
	select {
	case <-s.closeSignalCh:
		return ErrClosed
	default:
		return ctx.Err()
	}
}

// Close implements [io.Closer]. It stops the cleanup goroutine and marks the storage as closed.
// All subsequent method calls return [ErrClosed]. Safe to call more than once.
func (s *FS) Close() error {
	s.closeOnce.Do(func() { close(s.closeSignalCh) })
	<-s.closedCh

	return nil
}

// NewSession implements [SessionStorage].
func (s *FS) NewSession(
	ctx context.Context,
	sID string,
	response SessionResponse,
	ttl time.Duration,
) (_ *SessionMeta, outErr error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		now  = s.timeNow()
		hash = s.hash(sID)
	)

	// check if the session already exists - it may be expired, so we need to remove it first
	if meta, mErr := s.readSessionMetaH(hash); mErr == nil {
		if meta.IsExpired(now) { // session expired - remove it
			if err := s.root.RemoveAll(hash); err != nil {
				return nil, err
			}
		} else { // session exists and is not expired
			return nil, ErrSessionAlreadyExists
		}
	} else if !errors.Is(mErr, ErrSessionNotFound) { // unexpected error
		return nil, mErr
	}

	// create the deepest directory to automatically create all parent ones
	if err := s.root.MkdirAll(s.requestsDirH(hash), fsDirPerm); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	defer func() { // on any error (after creating any file/dir) remove the session dir to avoid leaving corrupted data
		if outErr != nil {
			_ = s.root.RemoveAll(hash) //nolint:errcheck
		}
	}()

	var expiresAt time.Time
	if ttl != NoExpiration {
		expiresAt = now.Add(ttl)
	}

	{ // create the session metadata file
		f, fErr := s.root.OpenFile(s.sessionMetaFileH(hash), os.O_CREATE|os.O_WRONLY|os.O_EXCL, fsFilePerm)
		if fErr != nil {
			return nil, fmt.Errorf("create session metadata file: %w", fErr)
		}

		if _, err := (&fsSessionMeta{
			CreatedAt: now,
			ExpiresAt: expiresAt,
			ID:        sID,
		}).WriteTo(f); err != nil {
			_ = f.Close()

			return nil, fmt.Errorf("write session metadata: %w", err)
		}

		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	{ // create the session data file
		approxCap := 64 + len(response.Headers)*8 //nolint:mnd // ~64 bytes of JSON overhead + ~8 bytes per header
		headers := make([]fsResponseHeader, len(response.Headers))

		for i, v := range response.Headers {
			headers[i] = fsResponseHeader(v)
			approxCap += len(v.Name) + len(v.Value)
		}

		var buf bytes.Buffer

		buf.Grow(approxCap + len(response.Body))

		if err := json.NewEncoder(&buf).Encode(fsSession{
			Code:      response.Code,
			Headers:   headers,
			Body:      response.Body,
			DelayNano: response.Delay.Nanoseconds(),
		}); err != nil {
			return nil, err
		}

		f, fErr := s.root.OpenFile(s.sessionDataFileH(hash), os.O_CREATE|os.O_WRONLY|os.O_EXCL, fsFilePerm)
		if fErr != nil {
			return nil, fmt.Errorf("create session data file: %w", fErr)
		}

		if err := s.compressor.Compress(&buf, f); err != nil {
			_ = f.Close()

			return nil, err
		}

		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	return &SessionMeta{
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

// GetSession implements [SessionStorage].
func (s *FS) GetSession(ctx context.Context, sID string) (*Session, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sHash := s.hash(sID)

	meta, mErr := s.readSessionMetaH(sHash)
	if mErr != nil {
		return nil, mErr
	}

	// don't remove expired sessions here - just return not found, and let the cleanup goroutine remove them
	// eventually, or once the client tries to create a new session with the same ID
	if meta.IsExpired(s.timeNow()) {
		return nil, ErrSessionNotFound
	}

	data, dErr := s.readSessionDataH(sHash)
	if dErr != nil {
		return nil, dErr
	}

	return &Session{Meta: meta.toSessionMeta(), Response: data.toSessionResponse()}, nil
}

// AddSessionTTL implements [SessionStorage].
func (s *FS) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		sHash = s.hash(sID)
		now   = s.timeNow()
	)

	meta, mErr := s.readSessionMetaH(sHash)
	if mErr != nil {
		return mErr
	}

	// expired = not found (creating a new session with the same ID will remove the expired one)
	if meta.IsExpired(now) {
		return ErrSessionNotFound
	}

	if howMuch == NoExpiration {
		meta.ExpiresAt = time.Time{}
	} else {
		meta.ExpiresAt = now.Add(howMuch)
	}

	f, fErr := s.root.OpenFile(s.sessionMetaFileH(sHash), os.O_WRONLY|os.O_TRUNC, fsFilePerm)
	if fErr != nil {
		return fErr
	}

	if _, err := meta.WriteTo(f); err != nil {
		_ = f.Close()

		return fmt.Errorf("write session metadata: %w", err)
	}

	return f.Close()
}

// DeleteSession implements [SessionStorage].
func (s *FS) DeleteSession(ctx context.Context, sID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sHash := s.hash(sID)

	meta, mErr := s.readSessionMetaH(sHash)
	if mErr != nil {
		return mErr
	}

	if meta.IsExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	if err := s.root.RemoveAll(sHash); err != nil {
		return fmt.Errorf("remove session directory: %w", err)
	}

	return nil
}

// NewRequest implements [RequestStorage].
func (s *FS) NewRequest( //nolint:funlen,gocognit
	ctx context.Context,
	sID, rID string,
	req CapturedRequest,
) (_ *RequestMeta, outErr error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		now          = s.timeNow()
		sHash, rHash = s.hash(sID), s.hash(rID)
	)

	sessMeta, mErr := s.readSessionMetaH(sHash)
	if mErr != nil {
		return nil, mErr
	}

	if sessMeta.IsExpired(now) {
		return nil, ErrSessionNotFound
	}

	// check the request existence before creating any files
	if _, statErr := s.root.Stat(s.requestMetaFileH(sHash, rHash)); statErr == nil {
		return nil, ErrRequestAlreadyExists
	}

	rIdx := newFSRequestIndex(s.root, s.requestsDirH(sHash))

	// evict the oldest requests if the per-session limit is reached
	if s.requestsLimit > 0 { //nolint:nestif
		var (
			list   = make([]sortEntry, 0, s.requestsLimit+1)
			idxErr error
		)

		if idxIter, idxIterErr := rIdx.ReadIndex(&idxErr); idxIterErr == nil {
			for idxEntry := range idxIter {
				list = append(list, sortEntry{id: idxEntry.Hash, createdAt: idxEntry.CreatedAt})
			}
		} else if !errors.Is(idxIterErr, os.ErrNotExist) {
			return nil, fmt.Errorf("read request index: %w", idxIterErr)
		}

		if idxErr != nil {
			return nil, fmt.Errorf("decode request index: %w", idxErr)
		}

		if uint(len(list)) >= s.requestsLimit {
			slices.SortFunc(list, func(a, b sortEntry) int { return b.createdAt.Compare(a.createdAt) })

			excess := list[s.requestsLimit-1:] // all entries at or beyond position limit-1

			for _, e := range excess {
				if err := s.root.Remove(s.requestMetaFileH(sHash, e.id)); err != nil && !errors.Is(err, os.ErrNotExist) {
					return nil, fmt.Errorf("remove request metadata: %w", err)
				}

				if err := s.root.Remove(s.requestDataFileH(sHash, e.id)); err != nil && !errors.Is(err, os.ErrNotExist) {
					return nil, fmt.Errorf("remove request data: %w", err)
				}
			}

			if err := rIdx.DeleteRecords(func(e fsRequestIndexRecord) bool {
				return slices.ContainsFunc(excess, func(ev sortEntry) bool { return e.Hash == ev.id })
			}); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("update request index after eviction: %w", err)
			}
		}
	}

	var (
		metaPath = s.requestMetaFileH(sHash, rHash)
		dataPath = s.requestDataFileH(sHash, rHash)
	)

	defer func() { // on any error after this point - remove new request files to avoid corrupted state
		if outErr != nil {
			_ = s.root.Remove(metaPath) //nolint:errcheck
			_ = s.root.Remove(dataPath) //nolint:errcheck
		}
	}()

	{ // write the request metadata file
		f, fErr := s.root.OpenFile(metaPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, fsFilePerm)
		if fErr != nil {
			return nil, fmt.Errorf("create request metadata file: %w", fErr)
		}

		if _, err := (&fsRequestMeta{
			CreatedAt: now,
			ID:        rID,
		}).WriteTo(f); err != nil {
			_ = f.Close()

			return nil, fmt.Errorf("write request metadata: %w", err)
		}

		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	{ // write the request data file
		approxCap := 64 + len(req.Headers)*8 //nolint:mnd // ~64 bytes of JSON overhead + ~8 bytes per header
		headers := make([]fsRequestHeader, len(req.Headers))

		for i, v := range req.Headers {
			headers[i] = fsRequestHeader(v)
			approxCap += len(v.Name) + len(v.Value)
		}

		var buf bytes.Buffer

		buf.Grow(approxCap + len(req.Body))

		if err := json.NewEncoder(&buf).Encode(fsRequest{
			ClientAddr: req.ClientAddr,
			Method:     req.Method,
			Body:       req.Body,
			Headers:    headers,
			URL:        req.URL,
		}); err != nil {
			return nil, err
		}

		f, fErr := s.root.OpenFile(dataPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, fsFilePerm)
		if fErr != nil {
			return nil, fmt.Errorf("create request data file: %w", fErr)
		}

		if err := s.compressor.Compress(&buf, f); err != nil {
			_ = f.Close()

			return nil, err
		}

		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	if err := rIdx.AddRecord(fsRequestIndexRecord{Hash: rHash, CreatedAt: now}); err != nil {
		return nil, fmt.Errorf("update request index: %w", err)
	}

	return &RequestMeta{CreatedAt: now}, nil
}

// GetRequest implements [RequestStorage].
func (s *FS) GetRequest(ctx context.Context, sID, rID string) (*Request, error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sHash, rHash := s.hash(sID), s.hash(rID)

	sessMeta, sErr := s.readSessionMetaH(sHash)
	if sErr != nil {
		return nil, sErr
	}

	if sessMeta.IsExpired(s.timeNow()) {
		return nil, ErrSessionNotFound
	}

	reqMeta, mErr := s.readRequestMetaH(sHash, rHash)
	if mErr != nil {
		return nil, mErr
	}

	data, dErr := s.readRequestDataH(sHash, rHash)
	if dErr != nil {
		return nil, dErr
	}

	return &Request{Meta: reqMeta.toRequestMeta(), Data: data.toCapturedRequest()}, nil
}

// GetRequests implements [RequestStorage].
func (s *FS) GetRequests( //nolint:gocognit,funlen
	ctx context.Context, sID string, iterErr *error,
) (iter.Seq2[string, Request], error) {
	if err := s.checkOpen(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()

	sHash := s.hash(sID)

	sessMeta, smErr := s.readSessionMetaH(sHash)
	if smErr != nil {
		s.mu.RUnlock()

		return nil, smErr
	}

	if sessMeta.IsExpired(s.timeNow()) {
		s.mu.RUnlock()

		return nil, ErrSessionNotFound
	}

	var (
		list   = make([]sortEntry, 0, 16) //nolint:mnd
		idxErr error
	)

	if idxIter, idxIterErr := newFSRequestIndex(s.root, s.requestsDirH(sHash)).ReadIndex(&idxErr); idxIterErr == nil {
		for entry := range idxIter {
			list = append(list, sortEntry{id: entry.Hash, createdAt: entry.CreatedAt})
		}
	} else if !errors.Is(idxIterErr, os.ErrNotExist) {
		s.mu.RUnlock()

		return nil, fmt.Errorf("read request index: %w", idxIterErr)
	}

	if idxErr != nil {
		s.mu.RUnlock()

		return nil, idxErr
	}

	s.mu.RUnlock()

	// sort requests by creation time, newest first (ID = hashed rID)
	slices.SortFunc(list, func(a, b sortEntry) int { return b.createdAt.Compare(a.createdAt) })

	return func(yield func(string, Request) bool) {
		for _, e := range list {
			s.mu.RLock()

			mf, mfErr := s.root.Open(s.requestMetaFileH(sHash, e.id))
			if mfErr != nil {
				s.mu.RUnlock()

				if errors.Is(mfErr, os.ErrNotExist) {
					continue // skip requests without metadata file (corrupted or in the middle of being created)
				}

				if iterErr != nil {
					*iterErr = mfErr // unexpected request metadata file error
				}

				return
			}

			var reqMeta fsRequestMeta

			_, mrErr := reqMeta.ReadFrom(mf)
			_ = mf.Close() // we don't need this file anymore

			// skip requests with unreadable metadata file, but report unexpected metadata file errors
			if mrErr != nil {
				s.mu.RUnlock()

				if iterErr != nil {
					*iterErr = mrErr
				}

				continue
			}

			// open request data file
			df, dfErr := s.root.Open(s.requestDataFileH(sHash, e.id))
			if dfErr != nil { // skip requests without data file, but report unexpected data file errors
				s.mu.RUnlock()

				if !errors.Is(dfErr, os.ErrNotExist) && iterErr != nil {
					*iterErr = dfErr

					return
				}

				continue
			}

			// prepare a reader for the request data file (decompress if needed)
			rc, rcErr := s.compressor.NewReader(df)
			if rcErr != nil {
				_ = df.Close()

				s.mu.RUnlock()

				if iterErr != nil {
					*iterErr = rcErr

					return // stop iteration: decompressor initialization failed
				}

				continue // no error sink - skip this entry and keep going
			}

			var data fsRequest

			decErr := s.decodeJSON(rc, &data)

			// checking the compressor error is required because the gzip compressor verifies the checksum when closing
			if cErr := rc.Close(); cErr != nil {
				_ = df.Close()

				if iterErr != nil {
					*iterErr = cErr
				}

				s.mu.RUnlock()

				continue // request data file is probably broken
			}

			_ = df.Close()

			s.mu.RUnlock()

			// decode the request data file into a request struct
			if decErr != nil {
				if iterErr != nil {
					*iterErr = fmt.Errorf("decode request data: %w", decErr)
				}

				continue
			}

			if !yield(reqMeta.ID, Request{Meta: reqMeta.toRequestMeta(), Data: data.toCapturedRequest()}) {
				return
			}
		}
	}, nil
}

// DeleteRequest implements [RequestStorage].
func (s *FS) DeleteRequest(ctx context.Context, sID, rID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sHash, rHash := s.hash(sID), s.hash(rID)

	sessMeta, smErr := s.readSessionMetaH(sHash)
	if smErr != nil {
		return smErr
	}

	if sessMeta.IsExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	requestMeta := s.requestMetaFileH(sHash, rHash)

	// check the request existence before attempting deletion
	if _, statErr := s.root.Stat(requestMeta); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return ErrRequestNotFound
		}

		return statErr
	}

	if err := s.root.Remove(requestMeta); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove request metadata: %w", err)
	}

	if err := s.root.Remove(s.requestDataFileH(sHash, rHash)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove request data: %w", err)
	}

	// update the request index - remove the entry for this request
	if err := newFSRequestIndex(s.root, s.requestsDirH(sHash)).DeleteRecords(func(e fsRequestIndexRecord) bool {
		return e.Hash == rHash
	}); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete request index entry: %w", err)
	}

	return nil
}

// DeleteAllRequests implements [RequestStorage].
func (s *FS) DeleteAllRequests(ctx context.Context, sID string) error {
	if err := s.checkOpen(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sHash := s.hash(sID)

	sessMeta, mErr := s.readSessionMetaH(sHash)
	if mErr != nil {
		return mErr
	}

	if sessMeta.IsExpired(s.timeNow()) {
		return ErrSessionNotFound
	}

	if err := s.root.RemoveAll(s.requestsDirH(sHash)); err != nil {
		return fmt.Errorf("remove requests directory: %w", err)
	}

	// recreate the directory so subsequent NewRequest calls can write into it
	if err := s.root.Mkdir(s.requestsDirH(sHash), fsDirPerm); err != nil {
		return fmt.Errorf("recreate requests directory: %w", err)
	}

	return nil
}

// hash returns the lowercase hex-encoded MD5 of hash of the input string.
// Used for generating filesystem paths.
func (*FS) hash(v string) string { h := md5.Sum([]byte(v)); return hex.EncodeToString(h[:]) } //nolint:nlreturn,gosec

// isValidHash reports whether v is a 32-character lowercase hex string (MD5 output).
// Used to distinguish session directories from other entries during directory scans.
func (*FS) isValidHash(v string) bool {
	const md5len = md5.Size * 2

	if len(v) != md5len {
		return false
	}

	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

// sessionMetaFileH returns the session metadata file path for a pre-hashed session ID.
//
// Example: "{sHash}/meta.bin".
func (*FS) sessionMetaFileH(sHash string) string { return path.Join(sHash, "meta.bin") }

// sessionDataFileH returns the session data file path for a pre-hashed session ID.
//
// Example: "{sHash}/session.json[.gz]".
func (s *FS) sessionDataFileH(sHash string) string {
	return path.Join(sHash, s.compressor.Filename("session.json"))
}

// requestsDirH returns the requests directory path for a pre-hashed session ID.
//
// Example: "{sHash}/requests".
func (*FS) requestsDirH(sHash string) string { return path.Join(sHash, "requests") }

// requestMetaFileH returns the request metadata file path for pre-hashed session and request IDs.
//
// Example: "{sHash}/requests/{rHash}.meta.bin".
func (s *FS) requestMetaFileH(sHash, rHash string) string {
	return path.Join(s.requestsDirH(sHash), rHash+requestMetaFilePostfix)
}

// requestDataFileH returns the request data file path for pre-hashed session and request IDs.
//
// Example: "{sHash}/requests/{rHash}.json[.gz]".
func (s *FS) requestDataFileH(sHash, rHash string) string {
	return path.Join(s.requestsDirH(sHash), s.compressor.Filename(rHash+".json"))
}

// requestHashFromFilename extracts the request hash from a request metadata filename such as
// "{hash}.meta.bin". Returns ("", false) if the name is not a valid request metadata filename.
func (s *FS) requestHashFromFilename(name string) (string, bool) {
	hash, ok := strings.CutSuffix(name, requestMetaFilePostfix)
	if !ok || !s.isValidHash(hash) {
		return "", false
	}

	return hash, true
}

// readSessionMetaH reads session metadata for a pre-hashed session ID. Returns [ErrSessionNotFound] if absent.
func (s *FS) readSessionMetaH(sHash string) (*fsSessionMeta, error) {
	f, fErr := s.root.Open(s.sessionMetaFileH(sHash))
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fErr
	}

	var meta fsSessionMeta

	if _, err := meta.ReadFrom(f); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("read session metadata: %w", err)
	}

	if err := f.Close(); err != nil {
		return nil, err
	}

	return &meta, nil
}

// decodeJSON reads r into a pooled buffer and unmarshals JSON into v.
func (s *FS) decodeJSON(r io.Reader, v any) error {
	buf := s.decBuf.Get()
	buf.Reset()

	defer s.decBuf.Put(buf)

	if _, err := io.Copy(buf, r); err != nil {
		return err
	}

	return json.Unmarshal(buf.Bytes(), v)
}

// readSessionDataH opens and decodes the session data file for sHash. Returns [ErrSessionNotFound] if absent.
func (s *FS) readSessionDataH(sHash string) (*fsSession, error) {
	f, fErr := s.root.Open(s.sessionDataFileH(sHash))
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fErr
	}

	rc, rcErr := s.compressor.NewReader(f)
	if rcErr != nil {
		_ = f.Close()

		return nil, rcErr
	}

	var data fsSession

	if err := s.decodeJSON(rc, &data); err != nil {
		_, _ = rc.Close(), f.Close()

		return nil, fmt.Errorf("decode session data: %w", err)
	}

	// checking the compressor error is required because the gzip compressor verifies the checksum when closing
	if err := rc.Close(); err != nil {
		_ = f.Close()

		return nil, err
	}

	if err := f.Close(); err != nil {
		return nil, err
	}

	return &data, nil
}

// readRequestMetaH opens and decodes the request metadata for sHash and rHash. Returns [ErrRequestNotFound] if absent.
func (s *FS) readRequestMetaH(sHash, rHash string) (*fsRequestMeta, error) {
	f, fErr := s.root.Open(s.requestMetaFileH(sHash, rHash))
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrRequestNotFound
		}

		return nil, fErr
	}

	var meta fsRequestMeta

	if _, err := meta.ReadFrom(f); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("read request metadata: %w", err)
	}

	if err := f.Close(); err != nil {
		return nil, err
	}

	return &meta, nil
}

// readRequestDataH opens and decodes the request data file for sHash and rHash. Returns [ErrRequestNotFound] if absent.
func (s *FS) readRequestDataH(sHash, rHash string) (*fsRequest, error) {
	f, fErr := s.root.Open(s.requestDataFileH(sHash, rHash))
	if fErr != nil {
		if errors.Is(fErr, os.ErrNotExist) {
			return nil, ErrRequestNotFound
		}

		return nil, fErr
	}

	rc, rcErr := s.compressor.NewReader(f)
	if rcErr != nil {
		_ = f.Close()

		return nil, rcErr
	}

	var data fsRequest

	if err := s.decodeJSON(rc, &data); err != nil {
		_, _ = rc.Close(), f.Close()

		return nil, fmt.Errorf("decode request data: %w", err)
	}

	// checking the compressor error is required because the gzip compressor verifies the checksum when closing
	if err := rc.Close(); err != nil {
		_ = f.Close()

		return nil, err
	}

	if err := f.Close(); err != nil {
		return nil, err
	}

	return &data, nil
}

// --------------------------------------------------------------------------------------------------------------------

// fsResponseHeader represents a single HTTP response header in the session data file.
type fsResponseHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// fsSession holds the response data stored in the session data file.
type fsSession struct {
	Code      uint16             `json:"code"`
	Headers   []fsResponseHeader `json:"headers,omitempty"`
	Body      []byte             `json:"body,omitempty"`
	DelayNano int64              `json:"delay_ns,omitempty"`
}

// toSessionResponse converts the internal session data to the public [SessionResponse] type.
func (d fsSession) toSessionResponse() SessionResponse {
	var headers []ResponseHeader
	if len(d.Headers) > 0 {
		headers = make([]ResponseHeader, len(d.Headers))
		for i, v := range d.Headers {
			headers[i] = ResponseHeader(v)
		}
	}

	return SessionResponse{Code: d.Code, Headers: headers, Body: d.Body, Delay: time.Duration(d.DelayNano)}
}

// fsRequestHeader represents a single HTTP request header in the request data file.
type fsRequestHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// fsRequest holds the captured request data stored in the request data file.
type fsRequest struct {
	ClientAddr string            `json:"client_addr,omitempty"`
	Method     string            `json:"method,omitempty"`
	Body       []byte            `json:"body,omitempty"`
	Headers    []fsRequestHeader `json:"headers,omitempty"`
	URL        string            `json:"url,omitempty"`
}

// toCapturedRequest converts the internal request data to the public [CapturedRequest] type.
func (d fsRequest) toCapturedRequest() CapturedRequest {
	var headers []RequestHeader
	if len(d.Headers) > 0 {
		headers = make([]RequestHeader, len(d.Headers))
		for i, v := range d.Headers {
			headers[i] = RequestHeader(v)
		}
	}

	return CapturedRequest{ClientAddr: d.ClientAddr, Method: d.Method, Body: d.Body, Headers: headers, URL: d.URL}
}

package frontend

import (
	"errors"
	"hash"
	"hash/fnv"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Handler is a static file server that serves files from a provided [fs.FS].
type Handler struct {
	root        fs.FS
	copyBufPool *sync.Pool
	hashPool    *sync.Pool
	etagCache   *lru[etagCacheKey, string]
}

type etagCacheKey struct {
	name    string
	size    int64
	modTime time.Time
}

// New returns an [http.Handler] that serves static files from root. It is designed for serving SPA (Single Page
// Application) frontends, so it has some specific behaviors:
//
//   - Existing files are served directly
//   - Any unrecognized path falls back to `index.html` with 200 (so the client-side router can take over)
//   - If `index.html` is absent, an embedded fallback 404 page is served instead
//
// Direct requests to `/index.html` are permanently redirected to `/` to avoid duplicate content.
//
// When the client sends `Accept-Encoding: gzip` and a pre-compressed `.gz` counterpart exists alongside the
// original (e.g. `app.js.gz` next to `app.js`), the compressed variant is served transparently with the appropriate
// `Content-Encoding` and `Vary` headers.
//
// ETags are computed via FNV-1a over the file content and stored in an LRU cache keyed by path, size and
// modification time, so repeated requests for the same file skip re-hashing entirely.
func New(root fs.FS) http.Handler {
	const etagCacheSize = 128

	return &Handler{
		root:        root,
		copyBufPool: &sync.Pool{New: func() any { return new(make([]byte, 32*1024)) }}, //nolint:mnd
		hashPool:    &sync.Pool{New: func() any { return fnv.New64a() }},
		etagCache:   newLRU[etagCacheKey, string](etagCacheSize),
	}
}

var _ http.Handler = (*Handler)(nil) // compile-time interface assertion

// ServeHTTP implements the [http.Handler] interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if the index file is requested directly, redirect to root to avoid duplicate content
	if r.URL.Path == "/index.html" {
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusMovedPermanently)

		return
	}

	// determine which file to serve based on the requested file path and open it
	f, filePath, fErr := h.fileToServe(r.URL.Path)
	if fErr != nil {
		http.Error(w, fErr.Error(), http.StatusInternalServerError) // unexpected error occurred

		return
	}

	defer func() { _ = f.Close() }()

	// determine and set the Content-Type header
	w.Header().Set("Content-Type", h.mimeType(f))

	// determine if client accepts gzip encoding AND a gzip version of the file exists - replace the file with its
	// gzip version
	if filePath != "" && clientAcceptsGzip(r) {
		if gz, ok := h.findGzipVersion(filePath); ok {
			_ = f.Close() // close the original file

			// set gzip-specific headers
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")

			f = gz // swap the file with its gzip version
		}
	}

	var (
		etag    string       // the ETag value to set in the response header
		etagKey etagCacheKey // the key to look up in the ETag cache
	)

	if stat, err := f.Stat(); err == nil {
		// set the last modified header if the file has a non-zero modification time
		if t := stat.ModTime(); !isZeroTime(t) {
			w.Header().Set("Last-Modified", t.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
		}

		// tell the client how big the response body will be
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

		etagKey = etagCacheKey{name: filePath, size: stat.Size(), modTime: stat.ModTime()}

		// try to get ETag from cache
		if v, ok := h.etagCache.Get(etagKey); ok {
			etag = v
		}
	}

	// if ETag is not cached, compute it and add to cache
	if etag == "" {
		etag = h.computeETag(f)

		if etag != "" && etagKey.name != "" {
			h.etagCache.Set(etagKey, etag) // put the computed ETag in cache
		}
	}

	// set the ETag header, if possible
	if etag != "" {
		w.Header().Set("ETag", `"`+etag+`"`)
	}

	// if the client's cached version is still valid, respond with 304 Not Modified without a body
	if etag != "" && filePath != "" && r.Header.Get("If-None-Match") == `"`+etag+`"` {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	if filePath != "" {
		w.WriteHeader(http.StatusOK) // set the status code to 200 OK
	} else {
		w.WriteHeader(http.StatusNotFound) // if filePath is empty, it means that we're serving the fallback 404 page
	}

	// for HEAD requests, we only need to send the headers without the body
	if r.Method == http.MethodHead {
		return
	}

	// copy the file content to the response body
	buf := h.copyBufPool.Get().(*[]byte) //nolint:errcheck,forcetypeassert
	defer h.copyBufPool.Put(buf)

	_, _ = io.CopyBuffer(w, f, *buf) //nolint:errcheck
}

// fileToServe decides which file to serve based on the requested file path. It returns:
//
//   - {[fs.File], string, nil} - if a file is found to serve
//   - {[fs.File], "", nil} - if the requested file is not found ([fs.File] will be the fallback 404 page)
//   - {nil, "", error} - if an unexpected error occurs while trying to open the file
//
// NOTE: The caller is responsible for closing the returned [fs.File].
func (h *Handler) fileToServe(filePath string) (fs.File, string, error) {
	const indexFileName = "index.html"

	// clean the file path and trim leading slashes to prevent directory traversal attacks and ensure consistent behavior
	filePath = strings.TrimLeft(path.Clean(filePath), "/")

	// if the cleaned file path is empty, serve the index file
	if filePath == "" {
		filePath = indexFileName
	}

	f, fErr := h.root.Open(filePath)
	if fErr != nil {
		if errors.Is(fErr, fs.ErrNotExist) { // requested file doesn't exist
			// try to open the index file (required for SPA to work coz any 404 => index.html)
			idx, idxErr := h.root.Open(indexFileName)
			if idxErr != nil {
				// index.html doesn't exist => fallback404.html (embedded in the binary), the filePath is set to empty string
				// to indicate that we're serving the fallback 404 page
				return newFallback404(), "", nil //nolint:nilerr
			}

			// index.html exists => return it (filePath is still the original requested path, not "index.html", because we
			// want to serve index.html for any non-existent path)
			return idx, filePath, nil
		}

		return nil, "", fErr // unexpected error
	}

	// requested file exists => return it
	return f, filePath, nil
}

// mimeType determines the MIME type of the given file. The logic is as follows:
//
//  1. Determine the MIME type based on the file extension using [mime.TypeByExtension].
//  2. Try to sniff the file's content using [http.DetectContentType] if the file implements [io.ReadSeeker].
//  3. If both methods fail, return "application/octet-stream" as a fallback.
func (h *Handler) mimeType(f fs.File) string {
	const fallback = "application/octet-stream"

	stat, statErr := f.Stat()
	if statErr != nil {
		return fallback
	}

	// first try to determine the content type from the file extension, if available
	if cType := mime.TypeByExtension(filepath.Ext(stat.Name())); cType != "" {
		return cType
	}

	// next, try to determine the content type by sniffing the file's content (requires io.ReadSeeker)
	if v, ok := f.(io.ReadSeeker); ok {
		var buf [512]byte // as used by http.DetectContentType

		n, rErr := io.ReadFull(v, buf[:])
		if rErr != nil && !errors.Is(rErr, io.ErrUnexpectedEOF) {
			return fallback
		}

		cType := http.DetectContentType(buf[:n])

		_, _ = v.Seek(0, io.SeekStart) //nolint:errcheck // reset file pointer

		return cType
	}

	return fallback
}

// findGzipVersion checks if a gzip version of the given file exists in the same directory. If it does, it returns
// the gzip file and true.
func (h *Handler) findGzipVersion(filePath string) (fs.File, bool) {
	if filePath == "" {
		return nil, false
	}

	f, gzErr := h.root.Open(filePath + ".gz")
	if gzErr != nil {
		return nil, false
	}

	return f, true
}

// computeETag computes a simple ETag for the given file by hashing its content. Provided file must implement
// [io.ReadSeeker].
//
// It returns an empty string if the file does not implement [io.ReadSeeker] or if any error occurs while
// reading/seeking the file.
func (h *Handler) computeETag(f fs.File) string {
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return ""
	}

	buf := h.copyBufPool.Get().(*[]byte) //nolint:errcheck,forcetypeassert
	defer h.copyBufPool.Put(buf)

	hsh := h.hashPool.Get().(hash.Hash64) //nolint:errcheck,forcetypeassert
	defer h.hashPool.Put(hsh)

	hsh.Reset()

	if _, err := io.CopyBuffer(hsh, rs, *buf); err != nil {
		return ""
	}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return ""
	}

	return strconv.FormatUint(hsh.Sum64(), 16)
}

// clientAcceptsGzip returns true if client accepts gzip-encoded responses.
func clientAcceptsGzip(r *http.Request) bool {
	for enc := range strings.SplitSeq(r.Header.Get("Accept-Encoding"), ",") {
		encName, params, _ := strings.Cut(strings.TrimSpace(enc), ";")

		if strings.TrimSpace(encName) != "gzip" {
			continue // not gzip, check next encoding
		}

		for param := range strings.SplitSeq(params, ";") {
			if after, ok := strings.CutPrefix(strings.TrimSpace(param), "q="); ok {
				q, err := strconv.ParseFloat(after, 64)
				if err == nil && q == 0 {
					return false
				}
			}
		}

		return true
	}

	return false
}

var unixEpochTime = time.Unix(0, 0) //nolint:gochecknoglobals

// isZeroTime checks if the given time is zero (uninitialized) or equal to the Unix epoch time.
func isZeroTime(t time.Time) bool { return t.IsZero() || t.Equal(unixEpochTime) }

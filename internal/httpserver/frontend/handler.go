package frontend

import (
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

//go:embed fallback404.html
var fallback404html []byte

// New creates new [http.Handler] that serves static files from the provided [fs.FS].
// If the requested file does not exist, it will try to serve "index.html" (required for SPA to work).
// It also supports serving pre-compressed gzip files if the client accepts gzip encoding and the .gz file exists.
func New(root fs.FS) http.Handler {
	var fileServer = http.FileServerFS(root)

	const (
		contentTypeHeader, contentTypeHTML = "Content-Type", "text/html; charset=utf-8"
		indexFileName                      = "index.html"
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var filePath = strings.TrimLeft(path.Clean(r.URL.Path), "/")

		if filePath == "" {
			filePath = indexFileName
		}

		fd, fErr := root.Open(filePath)
		switch { //nolint:wsl
		case os.IsNotExist(fErr): // if requested file does not exist
			index, indexErr := root.Open(indexFileName)
			if indexErr == nil { // always return index.html, if it exists (required for SPA to work)
				defer func() { _ = index.Close() }()

				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)

					return
				}

				w.Header().Set(contentTypeHeader, contentTypeHTML)
				w.WriteHeader(http.StatusOK)
				_, _ = io.Copy(w, index) //nolint:errcheck

				return
			}

			w.Header().Set(contentTypeHeader, contentTypeHTML)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(fallback404html) //nolint:errcheck

			return
		case fErr != nil: // some other error
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			http.Error(w, fmt.Errorf("failed to open file %s: %w", filePath, fErr).Error(), http.StatusInternalServerError)

			return
		}

		defer func() { _ = fd.Close() }()

		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)

			return
		}

		if clientAcceptsGzip(r.Header.Get("Accept-Encoding")) {
			if served := serveGzipIfExists(w, root, filePath); served {
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

// clientAcceptsGzip checks if the "Accept-Encoding" header indicates that the client accepts gzip encoding.
func clientAcceptsGzip(header string) bool {
	if header == "" {
		return false
	}

	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		var (
			encoding string
			q        = 1.0
		)

		if before, after, ok := strings.Cut(part, ";"); ok {
			encoding = strings.TrimSpace(before)

			for p := range strings.SplitSeq(after, ";") {
				p = strings.TrimSpace(p)

				if strings.HasPrefix(p, "q=") {
					v, err := strconv.ParseFloat(p[2:], 64)
					if err == nil {
						q = v
					}

					break
				}
			}
		} else {
			encoding = part
		}

		if encoding == "gzip" && q > 0 {
			return true
		}
	}

	return false
}

// serveGzipIfExists checks if a gzip version of the requested file exists and serves it if the client accepts
// gzip encoding.
// It returns true if the gzip file was served, false otherwise.
func serveGzipIfExists(w http.ResponseWriter, root fs.FS, filePath string) bool {
	gzPath := filePath + ".gz"

	fd, err := root.Open(gzPath)
	if err != nil {
		return false
	}

	defer func() { _ = fd.Close() }()

	// determine Content-Type from ORIGINAL filename
	if ct := mime.TypeByExtension(path.Ext(filePath)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	w.WriteHeader(http.StatusOK)

	_, _ = io.Copy(w, fd) //nolint:errcheck

	return true
}

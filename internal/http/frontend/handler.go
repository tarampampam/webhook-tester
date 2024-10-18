package frontend

import (
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed fallback404.html
var fallback404html []byte

func New(root fs.FS) http.Handler { //nolint:funlen
	var fileServer = http.FileServerFS(root)

	const (
		contentTypeHeader = "Content-Type"
		contentTypeHTML   = "text/html; charset=utf-8"
		indexFileName     = "index.html"
	)

	var fileCacheBoostExtensionsMap = map[string]struct{}{
		".gz":          {},
		".svg":         {},
		".png":         {},
		".jpg":         {},
		".jpeg":        {},
		".woff":        {},
		".otf":         {},
		".ttf":         {},
		".eot":         {},
		".ico":         {},
		".css":         {},
		".js":          {},
		".webmanifest": {},
	}

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
				_, _ = io.Copy(w, index)

				return
			}

			w.Header().Set(contentTypeHeader, contentTypeHTML)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(fallback404html)

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

		if ext := strings.ToLower(path.Ext(r.URL.Path)); ext != "" {
			if _, ok := fileCacheBoostExtensionsMap[ext]; ok {
				w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

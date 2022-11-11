package fileserver

import (
	"io"
	"net/http"
	"os"
	"path"
)

var fallback404 = []byte("<html><body><h1>Error 404</h1><h2>Not found</h2></body></html>") //nolint:gochecknoglobals

func NewHandler(root http.FileSystem) http.HandlerFunc {
	var (
		fileServer       = http.FileServer(root)
		errorPageContent []byte
	)

	if f, err := root.Open("404.html"); err == nil {
		errorPageContent, _ = io.ReadAll(f)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		f, err := root.Open(path.Clean(r.URL.Path))
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)

			if len(errorPageContent) > 0 {
				_, _ = w.Write(errorPageContent)
			} else {
				_, _ = w.Write(fallback404)
			}

			return
		}

		if err != nil {
			_ = f.Close()
		}

		fileServer.ServeHTTP(w, r)
	}
}

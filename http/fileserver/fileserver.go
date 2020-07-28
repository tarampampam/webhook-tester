package fileserver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"webhook-tester/http/errors"

	jsoniter "github.com/json-iterator/go"
)

// this HTML content will be used as fallback content for 404 response.
const fallback404content = "<html><body><h1>ERROR 404</h1><h2>Requested file was not found</h2></body></html>"

type (
	FileNotFoundHandler func(http.ResponseWriter, *http.Request)

	Settings struct {
		Root            http.Dir
		NotFoundHandler FileNotFoundHandler // optionally
		IndexFile       string
		Error404file    string
	}

	FileServer struct {
		Settings Settings
		json     jsoniter.API
	}
)

// NewFileServer creates new file server.
func NewFileServer(settings Settings) *FileServer {
	return &FileServer{
		Settings: settings,
		json:     jsoniter.ConfigFastest,
	}
}

// Serve requests to the "public" files and directories.
func (fileServer *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// redirect .../index.html to .../
	if strings.HasSuffix(r.URL.Path, "/"+fileServer.Settings.IndexFile) {
		http.Redirect(w, r, r.URL.Path[0:len(r.URL.Path)-len(fileServer.Settings.IndexFile)], http.StatusMovedPermanently)
		return
	}

	// if empty, set current directory
	dir := string(fileServer.Settings.Root)
	if dir == "" {
		dir = "."
	}

	// add prefix and clean
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	// add index file name if requested directory (or server root)
	if upath[len(upath)-1] == '/' {
		upath += fileServer.Settings.IndexFile
	}
	// make path clean
	upath = path.Clean(upath)

	// path to file
	name := path.Join(dir, filepath.FromSlash(upath))

	// if files server root directory is set - try to find file and serve them
	if len(fileServer.Settings.Root) > 0 {
		// check for file exists
		if f, err := os.Open(name); err == nil {
			// file exists and opened
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()
			// file (or directory) exists
			if stat, statErr := os.Stat(name); statErr == nil && stat.Mode().IsRegular() {
				// requested file is file (not directory)
				var modTime time.Time
				// Try to extract file modified time
				if info, err := f.Stat(); err == nil {
					modTime = info.ModTime()
				} else {
					modTime = time.Now() // fallback
				}
				// serve fie content
				http.ServeContent(w, r, filepath.Base(upath), modTime, f)

				return
			}
		}
	}

	// If all tries for content serving above has been failed - file was not found (HTTP 404)
	fileServer.handle404(w, r)
}

func (fileServer *FileServer) handle404(w http.ResponseWriter, r *http.Request) {
	if fileServer.Settings.NotFoundHandler != nil {
		// If "file not found" handler is set - call them
		fileServer.Settings.NotFoundHandler(w, r)

		return
	}

	// if JSON type was requested - respond using this format
	if strings.Contains(r.Header.Get("Accept"), "json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)

		_ = fileServer.json.NewEncoder(w).Encode(
			errors.NewServerError(http.StatusNotFound, "Not found"),
		)

		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// at first - we try to find local file with error content
	if len(fileServer.Settings.Root) > 0 {
		var errPage = string(fileServer.Settings.Root) + "/" + fileServer.Settings.Error404file
		if f, err := os.Open(errPage); err == nil {
			// file exists and opened
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()
			// file (or directory) exists
			if stat, statErr := os.Stat(errPage); statErr == nil && stat.Mode().IsRegular() {
				// requested file is file (not directory)
				if _, writeErr := io.Copy(w, f); writeErr != nil {
					panic(writeErr)
				}

				return
			}
		}
	}

	// fallback
	_, _ = fmt.Fprint(w, fallback404content)
}

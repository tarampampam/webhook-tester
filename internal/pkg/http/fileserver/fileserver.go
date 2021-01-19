// Package fileserver contains static files server implementation.
package fileserver

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	defaultFallbackErrorContent = "<html><body><h1>Error {{ code }}</h1><h2>{{ message }}</h2></body></html>"
	defaultIndexFileName        = "index.html"
)

// ErrorHandlerFunc is used as handler for errors processing. If func return `true` - next handler will be NOT executed.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) (doNotContinue bool)

// FileServer is a main file server structure (implements `http.Handler` interface).
type FileServer struct {
	// Server settings (some of them can be changed in runtime).
	Settings Settings

	// If all error handlers fails - this content will be used as fallback for error page generating.
	FallbackErrorContent string

	// Error handlers stack.
	ErrorHandlers []ErrorHandlerFunc
}

// Settings describes file server options.
type Settings struct {
	// Directory path, where files for serving is located.
	FilesRoot string

	// File name (relative path to the file) that will be used as an index (like <https://bit.ly/356QeFm>).
	IndexFileName string

	// File name (relative path to the file) that will be used as error page template.
	ErrorFileName string

	// Respond "index file" request with redirection to the root (`example.com/index.html` -> `example.com/`).
	RedirectIndexFileToRoot bool
}

// NewFileServer creates new file server with default settings. Feel free to change default behavior.
func NewFileServer(s Settings) (*FileServer, error) { //nolint:gocritic
	if info, err := os.Stat(s.FilesRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(`directory "%s" does not exists`, s.FilesRoot)
		}

		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf(`"%s" is not directory`, s.FilesRoot)
	}

	if s.IndexFileName == "" {
		s.IndexFileName = defaultIndexFileName
	}

	fs := &FileServer{
		Settings:             s,
		FallbackErrorContent: defaultFallbackErrorContent,
	}

	fs.ErrorHandlers = []ErrorHandlerFunc{
		JSONErrorHandler(),
		StaticHTMLPageErrorHandler(),
	}

	return fs, nil
}

func (fs *FileServer) handleError(w http.ResponseWriter, r *http.Request, errorCode int) {
	if fs.ErrorHandlers != nil && len(fs.ErrorHandlers) > 0 {
		for _, handler := range fs.ErrorHandlers {
			if handler(w, r, fs, errorCode) {
				return
			}
		}
	}

	// fallback
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(errorCode)

	_, _ = w.Write([]byte(ErrorPageTemplate(fs.FallbackErrorContent).Build(errorCode)))
}

// ServeHTTP responds to an HTTP request.
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fs.handleError(w, r, http.StatusMethodNotAllowed)

		return
	}

	if fs.Settings.RedirectIndexFileToRoot && len(fs.Settings.IndexFileName) > 0 {
		// redirect .../index.html to .../
		if strings.HasSuffix(r.URL.Path, "/"+fs.Settings.IndexFileName) {
			http.Redirect(w, r, r.URL.Path[0:len(r.URL.Path)-len(fs.Settings.IndexFileName)], http.StatusMovedPermanently)

			return
		}
	}

	urlPath := r.URL.Path

	// add leading `/` (if required)
	if urlPath == "" || !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + r.URL.Path
	}

	// if directory requested (or server root) - add index file name
	if len(fs.Settings.IndexFileName) > 0 && urlPath[len(urlPath)-1] == '/' {
		urlPath += fs.Settings.IndexFileName
	}

	// prepare target file path
	filePath := path.Join(fs.Settings.FilesRoot, filepath.FromSlash(path.Clean(urlPath)))

	// check for file existence
	if stat, err := os.Stat(filePath); err == nil && stat.Mode().IsRegular() {
		if file, err := os.Open(filePath); err == nil {
			defer func() { _ = file.Close() }()

			http.ServeContent(w, r, filepath.Base(filePath), stat.ModTime(), file)

			return
		}

		fs.handleError(w, r, http.StatusInternalServerError)

		return
	}

	fs.handleError(w, r, http.StatusNotFound)
}

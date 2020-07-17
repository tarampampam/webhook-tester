package fileserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileServer_ServeHTTP(t *testing.T) {
	t.Parallel()

	// Create directory in temporary
	createTempDir := func() string {
		t.Helper()

		if dir, err := ioutil.TempDir("", "test-"); err != nil {
			panic(err)
		} else {
			return dir
		}
	}

	tests := []struct {
		name                string
		giveDirs            []string
		giveFiles           map[string][]byte
		giveNotFoundHandler FileNotFoundHandler
		giveIndexFile       string
		giveError404file    string
		giveRequestURI      string
		giveRequestMethod   string
		giveRequestHeaders  map[string]string
		wantResponseCode    int
		wantResponseBody    []byte
		wantContentType     string
		wantRedirectTo      string
	}{
		{
			name: "Static TEXT file serving from local FS",
			giveFiles: map[string][]byte{
				"test1.txt": []byte("test content"),
			},
			giveRequestURI:    "/test1.txt",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusOK,
			wantResponseBody:  []byte("test content"),
			wantContentType:   "text/plain; charset=utf-8",
		},
		{
			name: "Static HTML file serving from local FS",
			giveFiles: map[string][]byte{
				"test1.html": []byte("<html>test content</html>"),
			},
			giveRequestURI:    "/test1.html",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusOK,
			wantResponseBody:  []byte("<html>test content</html>"),
			wantContentType:   "text/html; charset=utf-8",
		},
		{
			name:              "Redirect from .../index.html to .../",
			giveIndexFile:     "indx.html",
			giveRequestURI:    "/indx.html",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusMovedPermanently,
			wantRedirectTo:    "/",
		},
		{
			name:              "Redirect from .../index.html to .../ insime some directory",
			giveIndexFile:     "indx.html",
			giveRequestURI:    "/some/indx.html",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusMovedPermanently,
			wantRedirectTo:    "/some/",
		},
		{
			name: "Request root",
			giveFiles: map[string][]byte{
				"indx.html": []byte("test content"),
			},
			giveIndexFile:     "indx.html",
			giveRequestURI:    "",
			giveRequestMethod: "GET",
			wantResponseBody:  []byte("test content"),
			wantResponseCode:  http.StatusOK,
			wantContentType:   "text/html; charset=utf-8",
		},
		{
			name:     "Index file from some directory",
			giveDirs: []string{"foo"},
			giveFiles: map[string][]byte{
				"indx.html":                       []byte("index in root"),
				filepath.Join("foo", "indx.html"): []byte("index in foo"),
			},
			giveIndexFile:     "indx.html",
			giveRequestURI:    "/foo/",
			giveRequestMethod: "GET",
			wantResponseBody:  []byte("index in foo"),
			wantResponseCode:  http.StatusOK,
			wantContentType:   "text/html; charset=utf-8",
		},
		{
			name:     "404 on directory request",
			giveDirs: []string{"foo"},
			giveFiles: map[string][]byte{
				"indx.html":                       []byte("index in root"),
				filepath.Join("foo", "indx.html"): []byte("index in foo"),
			},
			giveIndexFile:     "indx.html",
			giveRequestURI:    "/foo",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusNotFound,
		},
		{
			name:              "NotFoundHandler handling",
			giveIndexFile:     "indx.html",
			giveRequestURI:    "/foo",
			giveRequestMethod: "GET",
			giveNotFoundHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(444)
				_, _ = w.Write([]byte("foo bar"))
				w.Header().Set("Content-Type", "blah blah")
			},
			wantResponseCode: 444,
			wantResponseBody: []byte("foo bar"),
			wantContentType:  "blah blah",
		},
		{
			name: "Error 404 file serving from local FS",
			giveFiles: map[string][]byte{
				"404.html": []byte("error 404 file"),
			},
			giveRequestURI:    "/foo",
			giveError404file:  "404.html",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusNotFound,
			wantResponseBody:  []byte("error 404 file"),
			wantContentType:   "text/html; charset=utf-8",
		},
		{
			name:               "Error 404 in json format",
			giveRequestURI:     "/foo",
			giveRequestMethod:  "GET",
			giveRequestHeaders: map[string]string{"accept": "application/json"},
			wantResponseCode:   http.StatusNotFound,
			wantResponseBody:   []byte(`{"error":true,"code":404,"message":"Not found"}` + "\n"),
			wantContentType:    "application/json; charset=utf-8",
		},
		{
			name:              "Error 404 fallback",
			giveRequestURI:    "/foo",
			giveError404file:  "404.html",
			giveRequestMethod: "GET",
			wantResponseCode:  http.StatusNotFound,
			wantResponseBody:  []byte("<html><body><h1>ERROR 404</h1><h2>Requested file was not found</h2></body></html>"),
			wantContentType:   "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var root http.Dir

			if len(tt.giveDirs) > 0 || len(tt.giveFiles) > 0 {
				tmpDir := createTempDir()
				root = http.Dir(tmpDir)

				defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

				// Create directories
				for _, d := range tt.giveDirs {
					assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
				}

				// Create files
				for name, content := range tt.giveFiles {
					if f, err := os.Create(filepath.Join(tmpDir, name)); err != nil {
						panic(err)
					} else {
						if _, err := f.Write(content); err != nil {
							panic(err)
						}
						if err := f.Close(); err != nil {
							panic(err)
						}
					}
				}
			} else {
				root = ""
			}

			fileServer := NewFileServer(Settings{
				Root:            root,
				NotFoundHandler: tt.giveNotFoundHandler,
				IndexFile:       tt.giveIndexFile,
				Error404file:    tt.giveError404file,
			})

			var (
				req, _ = http.NewRequest(tt.giveRequestMethod, tt.giveRequestURI, nil)
				rr     = httptest.NewRecorder()
			)

			for key, value := range tt.giveRequestHeaders {
				req.Header.Set(key, value)
			}

			fileServer.ServeHTTP(rr, req)

			assert.Equal(t, rr.Code, tt.wantResponseCode)

			if len(tt.wantResponseBody) > 0 && !reflect.DeepEqual(rr.Body.Bytes(), tt.wantResponseBody) {
				t.Errorf("Wrong HTTP response. Want [%s], got [%s]", tt.wantResponseBody, rr.Body.String())
			}

			if ct := rr.Header().Get("Content-Type"); tt.wantContentType != "" && ct != tt.wantContentType {
				t.Errorf("Wrong response content type header. Want %s, got %s", tt.wantContentType, ct)
			}

			if rt := rr.Header().Get("Location"); tt.wantRedirectTo != "" && tt.wantRedirectTo != rt {
				t.Errorf("Wrong redirect to location. Want %s, got %s", tt.wantRedirectTo, rt)
			}
		})
	}
}

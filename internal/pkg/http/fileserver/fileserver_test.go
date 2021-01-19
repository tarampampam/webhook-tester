package fileserver

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() { //nolint:gochecknoinits
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") //nolint:gochecknoglobals

func RandStringRunes(t *testing.T, n int) string {
	t.Helper()

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] //nolint:gosec
	}

	return string(b)
}

func TestNewFileServer_WrongDirectoryError(t *testing.T) {
	fs, err := NewFileServer(Settings{
		FilesRoot: RandStringRunes(t, 32),
	})

	assert.Nil(t, fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not exists")

	tmpDir, _ := ioutil.TempDir("", "test-")
	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)
	file, _ := os.Create(filepath.Join(tmpDir, "foo"))
	file.Close()

	fs, err = NewFileServer(Settings{
		FilesRoot: file.Name(),
	})

	assert.Nil(t, fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not directory")
}

func TestFileServer_ServeHTTP(t *testing.T) {
	var cases = []struct {
		name                   string
		giveDirs               []string
		giveFiles              map[string][]byte
		giveSettings           Settings
		giveRequestMethod      string
		giveRequestURI         string
		giveRequestHeaders     map[string]string
		beforeServing          func(fs *FileServer)
		wantResponseHTTPCode   int
		wantResponseContent    string
		wantResponseSubstrings []string
		resultCheckingFn       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:                   "serving request without URI",
			giveRequestURI:         "",
			wantResponseHTTPCode:   http.StatusNotFound,
			wantResponseSubstrings: []string{"Not Found"},
		},
		{
			name:           "static file serving",
			giveRequestURI: "/test",
			giveFiles: map[string][]byte{
				"test": []byte("test content"),
			},
			wantResponseHTTPCode: http.StatusOK,
			wantResponseContent:  "test content",
		},
		{
			name:           "static HTML file serving",
			giveRequestURI: "/test.html",
			giveFiles: map[string][]byte{
				"test.html": []byte("<p>test html content</p>"),
			},
			wantResponseHTTPCode: http.StatusOK,
			wantResponseContent:  "<p>test html content</p>",
			resultCheckingFn: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, rr.Header().Get("Content-Type"), "text/html; charset=utf-8")
			},
		},
		{
			name:                 "directory above (./../) requested",
			giveRequestURI:       "/../../../../etc/passwd",
			wantResponseHTTPCode: http.StatusNotFound,
		},
		{
			name: "disabled redirection from",
			giveSettings: Settings{
				IndexFileName: "idx.html",
			},
			giveRequestURI:       "/foo/idx.html",
			wantResponseHTTPCode: http.StatusNotFound,
		},
		{
			name: "redirect from ./{indexFileName} to ./",
			giveSettings: Settings{
				IndexFileName:           "idx.html",
				RedirectIndexFileToRoot: true,
			},
			giveRequestURI:       "/idx.html",
			wantResponseHTTPCode: http.StatusMovedPermanently,
			resultCheckingFn: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "/", rr.Header().Get("Location"))
			},
		},
		{
			name: "redirect from ./foo/{indexFileName} to ./foo/",
			giveSettings: Settings{
				IndexFileName:           "idx.html",
				RedirectIndexFileToRoot: true,
			},
			giveRequestURI:       "/foo/idx.html",
			wantResponseHTTPCode: http.StatusMovedPermanently,
			resultCheckingFn: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "/foo/", rr.Header().Get("Location"))
			},
		},
		{
			name: "index file in root directory serving",
			giveSettings: Settings{
				IndexFileName: "idx.html",
			},
			giveRequestURI: "/",
			giveFiles: map[string][]byte{
				"idx.html": []byte("index content"),
			},
			wantResponseHTTPCode: http.StatusOK,
			wantResponseContent:  "index content",
		},
		{
			name: "index file in sub-directory serving",
			giveSettings: Settings{
				IndexFileName: "idx.html",
			},
			giveRequestURI: "/foo/",
			giveDirs:       []string{"foo"},
			giveFiles: map[string][]byte{
				"idx.html":                       []byte("index in root"),
				filepath.Join("foo", "idx.html"): []byte("index in foo"),
			},
			wantResponseHTTPCode: http.StatusOK,
			wantResponseContent:  "index in foo",
		},
		{
			name: "404 on directory request",
			giveSettings: Settings{
				IndexFileName: "indx.html",
			},
			giveDirs: []string{"foo"},
			giveFiles: map[string][]byte{
				"indx.html":                       []byte("index in root"),
				filepath.Join("foo", "indx.html"): []byte("index in foo"),
			},
			giveRequestURI:       "/foo",
			wantResponseHTTPCode: http.StatusNotFound,
		},
		{
			name: "custom error handler",
			beforeServing: func(fs *FileServer) {
				fs.ErrorHandlers = []ErrorHandlerFunc{
					func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) bool {
						w.WriteHeader(444)
						_, _ = w.Write([]byte("foo bar"))
						w.Header().Set("Content-Type", "blah blah")

						return true
					},
				}
			},
			giveRequestURI:       "/foo",
			wantResponseHTTPCode: 444,
			wantResponseContent:  "foo bar",
			resultCheckingFn: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "blah blah", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "custom error handler fallback",
			beforeServing: func(fs *FileServer) {
				fs.ErrorHandlers = []ErrorHandlerFunc{
					func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) bool {
						return false
					},
				}
			},
			giveRequestURI:         "/foo",
			wantResponseHTTPCode:   http.StatusNotFound,
			wantResponseSubstrings: []string{"<html>", "Error 404", "Not Found", "</html>"},
		},
		{
			name:                 "error in json format when json requested",
			giveRequestURI:       "/foo",
			giveRequestHeaders:   map[string]string{"accept": "application/json"},
			wantResponseHTTPCode: http.StatusNotFound,
			resultCheckingFn: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.JSONEq(t, `{"code":404,"message":"Not Found"}`, rr.Body.String())
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
			assert.NoError(t, tmpDirErr)

			defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

			if len(tt.giveDirs) > 0 || len(tt.giveFiles) > 0 {
				for _, d := range tt.giveDirs {
					assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
				}

				for name, content := range tt.giveFiles {
					file, createErr := os.Create(filepath.Join(tmpDir, name))
					assert.NoError(t, createErr)
					_, fileWritingErr := file.Write(content)
					assert.NoError(t, fileWritingErr)
					assert.NoError(t, file.Close())
				}
			}

			if tt.giveSettings.FilesRoot == "" {
				tt.giveSettings.FilesRoot = tmpDir
			}

			fs, fsErr := NewFileServer(tt.giveSettings)

			assert.NoError(t, fsErr)

			var (
				req, _ = http.NewRequest(tt.giveRequestMethod, tt.giveRequestURI, nil)
				rr     = httptest.NewRecorder()
			)

			if tt.giveRequestHeaders != nil {
				for key, value := range tt.giveRequestHeaders {
					req.Header.Set(key, value)
				}
			}

			if tt.beforeServing != nil {
				tt.beforeServing(fs)
			}

			fs.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantResponseHTTPCode, rr.Code)

			if tt.wantResponseContent != "" {
				assert.Equal(t, tt.wantResponseContent, rr.Body.String())
			}

			if len(tt.wantResponseSubstrings) > 0 {
				for _, expected := range tt.wantResponseSubstrings {
					assert.Contains(t, rr.Body.String(), expected)
				}
			}

			if tt.resultCheckingFn != nil {
				tt.resultCheckingFn(t, rr)
			}
		})
	}
}

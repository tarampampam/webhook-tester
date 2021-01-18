package fileserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorPageTemplate_String(t *testing.T) {
	assert.Equal(t, "foo", ErrorPageTemplate("foo").String())
}

func TestErrorPageTemplate_Build(t *testing.T) {
	assert.Equal(t,
		"foo 200 <> OK",
		ErrorPageTemplate("foo {{ code }} <> {{ message }}").Build(200),
	)
}

func TestJSONErrorHandler(t *testing.T) {
	tmpDir, _ := ioutil.TempDir("", "test-")
	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

	fs, _ := NewFileServer(Settings{FilesRoot: tmpDir})
	assert.NotNil(t, fs)

	handler := JSONErrorHandler()

	var (
		req, _ = http.NewRequest(http.MethodGet, "", nil)
		rr     = httptest.NewRecorder()
	)

	assert.False(t, handler(rr, req, fs, http.StatusNotFound))

	req, _ = http.NewRequest(http.MethodGet, "", nil)
	req.Header.Add("Accept", "application/json")

	rr = httptest.NewRecorder()

	assert.True(t, handler(rr, req, fs, http.StatusNotFound))
	assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"code":404,"message":"Not Found"}`, rr.Body.String())
}

func TestStaticHtmlPageErrorHandler(t *testing.T) {
	tmpDir, _ := ioutil.TempDir("", "test-")
	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

	fs, _ := NewFileServer(Settings{
		FilesRoot: tmpDir,
	})
	assert.NotNil(t, fs)

	handler := StaticHTMLPageErrorHandler()

	var (
		req, _ = http.NewRequest(http.MethodGet, "", nil)
		rr     = httptest.NewRecorder()
	)

	assert.False(t, handler(rr, req, fs, http.StatusNotFound))

	// create template file
	file, _ := os.Create(filepath.Join(tmpDir, "error.html"))
	_, _ = file.Write([]byte("template: {{ message }} | {{ code }}"))
	file.Close()

	fs.Settings.ErrorFileName = "error.html"

	req, _ = http.NewRequest(http.MethodGet, "", nil)
	rr = httptest.NewRecorder()

	assert.True(t, handler(rr, req, fs, http.StatusBadGateway))
	assert.Equal(t, "text/html; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Equal(t, `template: Bad Gateway | 502`, rr.Body.String())

	rr = httptest.NewRecorder()

	assert.True(t, handler(rr, req, fs, http.StatusNotFound))
	assert.Equal(t, `template: Not Found | 404`, rr.Body.String())
}

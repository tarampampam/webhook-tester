package frontend_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/frontend"
)

func TestHandler(t *testing.T) {
	var (
		root = fs.FS(fstest.MapFS{
			"index.html": {
				Data: []byte("<html><body>index</body></html>"),
			},
			"404.html": {
				Data: []byte("<html><body>OLOLO 404</body></html>"),
			},
			"robots.txt": {
				Data: []byte("User-agent: *\nDisallow: /"),
			},
		})
	)

	for name, tt := range map[string]struct {
		giveUrl               string
		giveMethod            string
		wantCode              int
		wantInBody            string
		wantEmptyResponseBody bool
	}{
		"root": {
			giveUrl:    "/",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<body>index</body>",
		},
		"root (head)": {
			giveUrl:               "/",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
		"index": {
			giveUrl:               "/index.html",
			giveMethod:            http.MethodGet,
			wantCode:              http.StatusMovedPermanently,
			wantEmptyResponseBody: true,
		},
		"not found": {
			giveUrl:    "/foo",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<html><body>index</body></html>",
		},
		"not found (head)": {
			giveUrl:               "/foo",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
		"existing file (head)": {
			giveUrl:               "/robots.txt",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var (
				req = httptest.NewRequest(tt.giveMethod, tt.giveUrl, http.NoBody)
				rec = httptest.NewRecorder()
			)

			frontend.New(root).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantEmptyResponseBody {
				assert.Empty(t, rec.Body.String())
			} else {
				assert.Contains(t, rec.Body.String(), tt.wantInBody)
			}
		})
	}
}

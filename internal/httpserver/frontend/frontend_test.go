package frontend_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/frontend"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	var root = fs.FS(fstest.MapFS{
		"index.html": {
			Data: []byte("<html><body>index</body></html>"),
		},
		"index.html.gz": {
			Data: []byte("GZIP_INDEX"),
		},
		"robots.txt": {
			Data: []byte("User-agent: *\nDisallow: /"),
		},
		"robots.txt.gz": {
			Data: []byte("GZIP_ROBOTS"),
		},
	})

	for name, tt := range map[string]struct {
		giveURL               string
		giveMethod            string
		giveHeaders           map[string]string
		wantCode              int
		wantInBody            string
		wantEmptyResponseBody bool
		wantHeaders           map[string]string
	}{
		// basic behavior
		"root": {
			giveURL:    "/",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<body>index</body>",
		},
		"root (head)": {
			giveURL:               "/",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
		"index redirect": {
			giveURL:               "/index.html",
			giveMethod:            http.MethodGet,
			wantCode:              http.StatusMovedPermanently,
			wantEmptyResponseBody: true,
		},
		"not found -> SPA fallback": {
			giveURL:    "/foo",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<html><body>index</body></html>",
		},
		"not found (head)": {
			giveURL:               "/foo",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
		"existing file (head)": {
			giveURL:               "/robots.txt",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},

		// gzip behavior
		"gzip index served": {
			giveURL:    "/",
			giveMethod: http.MethodGet,
			giveHeaders: map[string]string{
				"Accept-Encoding": "gzip",
			},
			wantCode:   http.StatusOK,
			wantInBody: "GZIP_INDEX",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/html; charset=utf-8",
			},
		},
		"gzip robots served": {
			giveURL:    "/robots.txt",
			giveMethod: http.MethodGet,
			giveHeaders: map[string]string{
				"Accept-Encoding": "gzip",
			},
			wantCode:   http.StatusOK,
			wantInBody: "GZIP_ROBOTS",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/plain; charset=utf-8",
			},
		},
		"gzip HEAD": {
			giveURL:    "/robots.txt",
			giveMethod: http.MethodHead,
			giveHeaders: map[string]string{
				"Accept-Encoding": "gzip",
			},
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
		},
		"gzip disabled via q=0": {
			giveURL:    "/robots.txt",
			giveMethod: http.MethodGet,
			giveHeaders: map[string]string{
				"Accept-Encoding": "gzip;q=0",
			},
			wantCode:   http.StatusOK,
			wantInBody: "User-agent",
		},
		"gzip fallback when encoding unsupported": {
			giveURL:    "/robots.txt",
			giveMethod: http.MethodGet,
			giveHeaders: map[string]string{
				"Accept-Encoding": "br",
			},
			wantCode:   http.StatusOK,
			wantInBody: "User-agent",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.giveMethod, tt.giveURL, http.NoBody)

			for k, v := range tt.giveHeaders {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()

			frontend.New(root).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)

			// --- body checks
			if tt.wantEmptyResponseBody {
				assert.Empty(t, rec.Body.String())
			} else if tt.wantInBody != "" {
				assert.Contains(t, rec.Body.String(), tt.wantInBody)
			}

			// --- header checks
			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, rec.Header().Get(k))
			}
		})
	}
}

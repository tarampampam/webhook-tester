package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestNewCacheControl(t *testing.T) {
	t.Parallel()

	const prefix = "http://localhost"

	t.Run("common cases", func(t *testing.T) {
		t.Parallel()

		for _, tt := range [...]struct {
			giveURL         string
			wantCacheHeader bool
		}{
			{"/", false},
			{"/abc", false},
			{"/bc?param=1", false},
			{"/awd?param=file.gz", false},

			{"/a.gz", true},
			{"////foo/a.svg", true},
			{"/a.png", true},
			{"/a.jpg", true},
			{"/a.jpeg", true},
			{"/a/b.woff", true},
			{"/a.otf", true},
			{"/a.ttf", true},
			{"/a.eot", true},
			{"/a.ico", true},
			{"/a.css", true},
			{"/a.js", true},
			{"/site.webmanifest", true},
		} {
			t.Run(tt.giveURL, func(t *testing.T) {
				t.Parallel()

				var (
					rr       = httptest.NewRecorder()
					req      = httptest.NewRequest(http.MethodGet, prefix+tt.giveURL, nil)
					executed = false
				)

				middleware.NewCacheControl(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executed = true
				})).ServeHTTP(rr, req)

				assert.True(t, executed)

				if tt.wantCacheHeader {
					assert.Equal(t, "public, max-age=604800", rr.Header().Get("Cache-Control"))
				} else {
					assert.Empty(t, rr.Header().Get("Cache-Control"))
				}
			})
		}
	})

	t.Run("skipper", func(t *testing.T) {
		t.Parallel()

		var (
			rr       = httptest.NewRecorder()
			req      = httptest.NewRequest(http.MethodGet, prefix+"/a.js", nil)
			executed = false
			skipper  = func(r *http.Request) bool { return strings.HasSuffix(r.URL.Path, ".js") }
		)

		middleware.NewCacheControl(skipper)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executed = true
		})).ServeHTTP(rr, req)

		assert.True(t, executed)
		assert.Empty(t, rr.Header().Get("Cache-Control"))
	})
}

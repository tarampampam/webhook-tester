package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestPreCleanPath(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		givePath   string
		giveRawURI string
		wantPath   string
	}{
		"already clean path is unchanged": {
			givePath:   "/foo/bar",
			giveRawURI: "/foo/bar",
			wantPath:   "/foo/bar",
		},
		"root path is unchanged": {
			givePath:   "/",
			giveRawURI: "/",
			wantPath:   "/",
		},
		"repeated slashes collapsed": {
			givePath:   "/foo////bar",
			giveRawURI: "/foo////bar",
			wantPath:   "/foo/bar",
		},
		"leading double slash collapsed": {
			givePath:   "//foo/bar",
			giveRawURI: "//foo/bar",
			wantPath:   "/foo/bar",
		},
		"trailing slash removed": {
			givePath:   "/foo/bar/",
			giveRawURI: "/foo/bar/",
			wantPath:   "/foo/bar",
		},
		"dot segment resolved": {
			givePath:   "/foo/../bar",
			giveRawURI: "/foo/../bar",
			wantPath:   "/bar",
		},
		"raw URI preserved when path is cleaned": {
			givePath:   "/foo////bar////////baz",
			giveRawURI: "/foo////bar////////baz?x=1",
			wantPath:   "/foo/bar/baz",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				capturedPath   string
				capturedRawURI string
			)

			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedRawURI = r.RequestURI
			})

			req := httptest.NewRequest(http.MethodGet, tc.givePath, http.NoBody)
			req.RequestURI = tc.giveRawURI

			middleware.PreCleanPath(next).ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tc.wantPath, capturedPath)
			assert.Equal(t, tc.giveRawURI, capturedRawURI) // RequestURI must never be modified
		})
	}
}

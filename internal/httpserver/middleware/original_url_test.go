package middleware_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestNewOriginalURL(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		giveRequestURI    string
		giveHost          string
		giveTLS           bool
		giveDoubleWrapped bool
		wantURL           string
	}{
		"http with simple path": {
			giveRequestURI: "/foo/bar",
			giveHost:       "example.com",
			wantURL:        "http://example.com/foo/bar",
		},
		"http with repeated slashes preserved": {
			giveRequestURI: "/foo////bar////////baz",
			giveHost:       "example.com",
			wantURL:        "http://example.com/foo////bar////////baz",
		},
		"query string preserved": {
			giveRequestURI: "/foo////bar?yes=no&x=1",
			giveHost:       "example.com",
			wantURL:        "http://example.com/foo////bar?yes=no&x=1",
		},
		"https when TLS is set": {
			giveRequestURI: "/secure/path",
			giveHost:       "secure.example.com",
			giveTLS:        true,
			wantURL:        "https://secure.example.com/secure/path",
		},
		"first middleware wins when applied twice": {
			giveRequestURI:    "/original",
			giveHost:          "example.com",
			giveDoubleWrapped: true,
			wantURL:           "http://example.com/original",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var capturedURL string

			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				u, ok := middleware.OriginalURLFromContext(r.Context())
				assert.True(t, ok)

				capturedURL = u.String()
			})

			var handler http.Handler
			if tc.giveDoubleWrapped {
				handler = middleware.NewOriginalURL()(middleware.NewOriginalURL()(next))
			} else {
				handler = middleware.NewOriginalURL()(next)
			}

			req := httptest.NewRequest(http.MethodGet, tc.giveRequestURI, http.NoBody)
			req.Host = tc.giveHost
			req.RequestURI = tc.giveRequestURI

			if tc.giveTLS {
				req.TLS = &tls.ConnectionState{}
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tc.wantURL, capturedURL)
		})
	}
}

func TestOriginalURLFromContext_NotSet(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	u, ok := middleware.OriginalURLFromContext(req.Context())

	assert.Equal(t, false, ok)
	assert.Nil(t, u)
}

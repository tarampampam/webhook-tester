package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestApply(t *testing.T) {
	t.Parallel()

	t.Run("common cases", func(t *testing.T) {
		t.Parallel()

		var calls []string

		mw := func(name string) func(http.Handler) http.Handler {
			return func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls = append(calls, "before-"+name)

					next.ServeHTTP(w, r)

					calls = append(calls, "after-"+name)
				})
			}
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "handler")
		})

		h := middleware.Apply(
			handler,
			mw("A"),
			mw("B"),
			mw("C"),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.DeepEqual(t,
			[]string{
				"before-A",
				"before-B",
				"before-C",
				"handler",
				"after-C",
				"after-B",
				"after-A",
			},
			calls,
		)
	})

	t.Run("skip nil middleware", func(t *testing.T) {
		t.Parallel()

		var called bool

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		h := middleware.Apply(
			handler,
			nil,
			nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, called)
	})

	t.Run("no middleware", func(t *testing.T) {
		t.Parallel()

		var called bool

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		h := middleware.Apply(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, called)
	})

	t.Run("modify request", func(t *testing.T) {
		t.Parallel()

		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Test", "123")
				next.ServeHTTP(w, r)
			})
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "123", r.Header.Get("X-Test"))
		})

		h := middleware.Apply(handler, mw)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
	})
}

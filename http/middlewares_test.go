package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisableCachingMiddleware(t *testing.T) {
	var handled bool = false

	// create a handler to use as "next" which will verify the request
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
			t.Error("Wrong header `Cache-Control` value found")
		}
		if w.Header().Get("Pragma") != "no-cache" {
			t.Error("Wrong header `Pragma` value found")
		}
		if w.Header().Get("Expires") != "0" {
			t.Error("Wrong header `Expires` value found")
		}

		handled = true
	})

	middlewareHandler := DisableCachingMiddleware(nextHandler)

	var (
		req, _ = http.NewRequest("GET", "http://testing", nil)
		rr     = httptest.NewRecorder()
	)

	if rr.Header().Get("Cache-Control") != "" {
		t.Error("Header `Cache-Control` must be empty before execution")
	}

	if rr.Header().Get("Pragma") != "" {
		t.Error("Header `Pragma` must be empty before execution")
	}

	if rr.Header().Get("Expires") != "" {
		t.Error("Header `Expires` must be empty before execution")
	}

	middlewareHandler.ServeHTTP(rr, req)

	if handled != true {
		t.Error("next handler was not executed")
	}
}

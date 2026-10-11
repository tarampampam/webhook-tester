package middleware

import (
	"net/http"
	"path"
)

// PreCleanPath is a middleware that normalizes r.URL.Path by collapsing repeated slashes and resolving dot
// segments before the request reaches [http.ServeMux].
//
// Why this is needed: Go's [http.ServeMux] unconditionally calls [path.Clean] on every incoming URL. When the
// cleaned path differs from the original (e.g. "////bar" → "/bar"), ServeMux issues a 307 Temporary Redirect
// to the cleaned path before any handler runs. The redirect goes out to the client, which then sends a new
// HTTP request with the already-clean URL - the original URL is gone.
//
// This middleware pre-cleans r.URL.Path so that ServeMux sees an already-clean path and skips the redirect.
// r.RequestURI is intentionally left untouched - it holds the raw HTTP request-line URI and is used by
// [NewOriginalURL] to capture the original URL before this middleware runs.
//
// See also: https://github.com/golang/go/issues/19481
func PreCleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clean := path.Clean(r.URL.Path); clean != r.URL.Path {
			r = r.Clone(r.Context())
			r.URL.Path = clean
		}

		next.ServeHTTP(w, r)
	})
}

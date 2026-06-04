package middleware

import (
	"context"
	"net/http"
	"net/url"
)

// originalUrlOnceCtxKey is a context key for the duplicate-injection guard in [NewOriginalURL].
type originalUrlOnceCtxKey struct{}

// originalURLCtxKey is a context key for the original request URL stored by [NewOriginalURL].
type originalURLCtxKey struct{}

// NewOriginalURL returns a middleware that captures the full original request URL into the context before any
// other middleware or handler processes the request. Use [OriginalURLFromContext] to retrieve it.
//
// The URL is built from the request scheme, host, and raw request URI (r.RequestURI), so it reflects the
// exact URL the client sent - including repeated slashes, dot segments, or any other un-normalized path.
//
// The reason - https://github.com/golang/go/issues/19481
//
// It is safe to use this middleware multiple times in the chain - the inner invocations are no-ops thanks to
// the duplicate-injection guard.
func NewOriginalURL() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(originalUrlOnceCtxKey{}) != nil {
				next.ServeHTTP(w, r) // already invoked, pass through

				return
			}

			ctx := context.WithValue(r.Context(), originalUrlOnceCtxKey{}, struct{}{})

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			u, err := r.URL.Parse(scheme + "://" + r.Host + r.RequestURI)
			if err != nil {
				u = r.URL // fallback to the original URL if parsing fails
			}

			ctx = context.WithValue(ctx, originalURLCtxKey{}, u)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OriginalURLFromContext retrieves the original request URL stored by [NewOriginalURL].
// Returns false if the middleware was not applied to the request.
func OriginalURLFromContext(ctx context.Context) (*url.URL, bool) {
	if v, ok := ctx.Value(originalURLCtxKey{}).(*url.URL); ok {
		return v, true
	}

	return nil, false
}

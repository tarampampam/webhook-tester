package middleware

import (
	"net/http"
	"path"
	"strings"
)

// NewCacheControl returns a middleware that sets Cache-Control header for static files.
//
// To skip the middleware for some requests, you can provide a skipper function. If the skipper returns true,
// the middleware will be skipped for that request.
func NewCacheControl(
	skipper func(*http.Request) bool, // optional, may be nil
) func(http.Handler) http.Handler {
	// the list of extensions for which the cache control header should be set
	fileExtensionsMap := map[string]struct{}{
		".gz":          {},
		".svg":         {},
		".png":         {},
		".jpg":         {},
		".jpeg":        {},
		".woff":        {},
		".otf":         {},
		".ttf":         {},
		".eot":         {},
		".ico":         {},
		".css":         {},
		".js":          {},
		".webmanifest": {},
	}

	const headerName = "Cache-Control"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipper != nil && skipper(r) {
				next.ServeHTTP(w, r)

				return
			}

			// doubled middleware invocation guard - skip the middleware if the request already has the Cache-Control
			// header set
			if w.Header().Get(headerName) != "" {
				next.ServeHTTP(w, r)

				return
			}

			if ext := strings.ToLower(path.Ext(r.URL.Path)); ext != "" {
				if _, ok := fileExtensionsMap[ext]; ok {
					w.Header().Set(headerName, "public, max-age=604800") // 1 week
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

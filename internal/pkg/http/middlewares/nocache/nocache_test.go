package nocache

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	var (
		req, _  = http.NewRequest(http.MethodGet, "http://testing", nil)
		rr      = httptest.NewRecorder()
		handled bool
	)

	assert.Empty(t, rr.Header().Get("Cache-Control"))
	assert.Empty(t, rr.Header().Get("Pragma"))
	assert.Empty(t, rr.Header().Get("Expires"))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
		assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
		assert.Equal(t, "0", w.Header().Get("Expires"))

		handled = true
	})

	New().Middleware(nextHandler).ServeHTTP(rr, req)

	assert.True(t, handled)
}

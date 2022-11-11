package nocache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/nocache"
)

func TestMiddleware(t *testing.T) {
	var (
		req, _  = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
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

	nocache.New().Middleware(nextHandler).ServeHTTP(rr, req)

	assert.True(t, handled)
}

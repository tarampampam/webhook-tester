package json_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/json"
)

func TestMiddleware(t *testing.T) {
	var (
		req, _  = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
		rr      = httptest.NewRecorder()
		handled bool
	)

	assert.Empty(t, rr.Header().Get("Content-Type"))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		handled = true
	})

	json.New().Middleware(nextHandler).ServeHTTP(rr, req)

	assert.True(t, handled)
}

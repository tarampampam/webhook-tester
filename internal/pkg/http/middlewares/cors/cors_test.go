package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/cors"

	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	var (
		req, _  = http.NewRequest(http.MethodGet, "http://testing", nil)
		rr      = httptest.NewRecorder()
		handled bool
	)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

		handled = true
	})

	cors.New().Middleware(nextHandler).ServeHTTP(rr, req)

	assert.True(t, handled)
}

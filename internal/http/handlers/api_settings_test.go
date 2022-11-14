package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/webhook-tester/internal/config"
	"github.com/tarampampam/webhook-tester/internal/http/handlers"
)

func TestApiSettings_ApiSettings(t *testing.T) {
	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{
				MaxRequests:        123,
				SessionTTL:         time.Second * 321,
				MaxRequestBodySize: 222,
			}, nil, nil, nil, nil, nil, "", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.ApiSettings(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.JSONEq(t, `{
		"limits": {"max_requests":123, "session_lifetime_sec":321, "max_webhook_body_size": 222}
	}`, rr.Body.String())
}

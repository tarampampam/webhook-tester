package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/webhook-tester/internal/config"
	"github.com/tarampampam/webhook-tester/internal/http/handlers"
)

func TestApiVersion_ApiAppVersion(t *testing.T) {
	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{}, nil, nil, nil, nil, nil, "1.2.3@foo", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.ApiAppVersion(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.JSONEq(t, `{"version":"1.2.3@foo"}`, rr.Body.String())
}

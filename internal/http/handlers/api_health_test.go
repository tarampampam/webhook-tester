package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tarampampam/webhook-tester/internal/http/handlers"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
)

func TestApiHealth_LivenessProbe(t *testing.T) {
	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{}, nil, nil, nil, nil, nil, "", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.LivenessProbe(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Empty(t, rr.Body.String())
}

func TestApiHealth_ReadinessProbe(t *testing.T) {
	// start mini-redis
	mini, err := miniredis.Run()
	require.NoError(t, err)

	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})

	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{}, rdb, nil, nil, nil, nil, "", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.ReadinessProbe(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Empty(t, rr.Body.String())
}

func TestApiHealth_ReadinessProbeFail(t *testing.T) {
	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{}, redis.NewClient(&redis.Options{}), nil, nil, nil, nil, "", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.ReadinessProbe(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusServiceUnavailable)
	assert.Contains(t, rr.Body.String(), "connection refused")
}

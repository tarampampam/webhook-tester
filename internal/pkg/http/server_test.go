package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func getRandomTCPPort(t *testing.T) (int, error) {
	t.Helper()

	// zero port means randomly (os) chosen port
	l, err := net.Listen("tcp", ":0") //nolint:gosec
	if err != nil {
		return 0, err
	}

	port := l.Addr().(*net.TCPAddr).Port

	if closingErr := l.Close(); closingErr != nil {
		return 0, closingErr
	}

	return port, nil
}

func checkTCPPortIsBusy(t *testing.T, port int) bool {
	t.Helper()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return true
	}

	_ = l.Close()

	return false
}

func TestServer_StartAndStop(t *testing.T) {
	port, err := getRandomTCPPort(t)
	assert.NoError(t, err)

	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	srv := NewServer(zap.NewNop())

	assert.False(t, checkTCPPortIsBusy(t, port))

	go func() {
		startingErr := srv.Start("", uint16(port))

		if !errors.Is(startingErr, http.ErrServerClosed) {
			assert.NoError(t, startingErr)
		}
	}()

	for i := 0; ; i++ {
		if !checkTCPPortIsBusy(t, port) {
			if i > 100 {
				t.Error("too many attempts for server start checking")
			}

			<-time.After(time.Microsecond * 10)
		} else {
			break
		}
	}

	assert.True(t, checkTCPPortIsBusy(t, port))
	assert.NoError(t, srv.Stop(context.Background()))
	assert.False(t, checkTCPPortIsBusy(t, port))
}

func TestServer_Register(t *testing.T) {
	uuid := "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	var routes = []struct {
		name    string
		route   string
		methods []string
	}{
		{name: "api_settings_get", route: "/api/settings", methods: []string{http.MethodGet}},
		{name: "api_get_version", route: "/api/version", methods: []string{http.MethodGet}},
		{name: "api_session_create", route: "/api/session", methods: []string{http.MethodPost}},
		{name: "api_session_delete", route: "/api/session/{sessionUUID:" + uuid + "}", methods: []string{http.MethodDelete}},                                             //nolint:lll
		{name: "api_session_requests_all_get", route: "/api/session/{sessionUUID:" + uuid + "}/requests", methods: []string{http.MethodGet}},                             //nolint:lll
		{name: "api_session_request_get", route: "/api/session/{sessionUUID:" + uuid + "}/requests/{requestUUID:" + uuid + "}", methods: []string{http.MethodGet}},       //nolint:lll
		{name: "api_delete_session_request", route: "/api/session/{sessionUUID:" + uuid + "}/requests/{requestUUID:" + uuid + "}", methods: []string{http.MethodDelete}}, //nolint:lll
		{name: "api_delete_all_session_requests", route: "/api/session/{sessionUUID:" + uuid + "}/requests", methods: []string{http.MethodDelete}},                       //nolint:lll
		{
			name:  "webhook",
			route: "/{sessionUUID:" + uuid + "}",
			methods: []string{
				http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
				http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
			},
		},
		{
			name:  "webhook_with_status_code",
			route: "/{sessionUUID:" + uuid + "}/{statusCode:[1-5][0-9][0-9]}",
			methods: []string{
				http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
				http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
			},
		},
		{
			name:  "webhook_any",
			route: "/{sessionUUID:" + uuid + "}/{any:.*}",
			methods: []string{
				http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
				http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
			},
		},
		{name: "metrics", route: "/metrics", methods: []string{http.MethodGet}},
		{name: "ready", route: "/ready", methods: []string{http.MethodGet, http.MethodHead}},
		{name: "live", route: "/live", methods: []string{http.MethodGet, http.MethodHead}},
		{name: "static", route: "/", methods: []string{http.MethodGet, http.MethodHead}},
		{name: "ws_session", route: "/ws/session/{sessionUUID:" + uuid + "}", methods: []string{http.MethodGet}},
	}

	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	srv := NewServer(zap.NewNop())

	router := srv.router // dirty hack, yes, i know

	for _, r := range routes {
		assert.Nil(t, router.Get(r.name))
	}

	stor := storage.NewInMemory(time.Second, 16)
	defer func() { _ = stor.Close() }()

	pubSub := pubsub.NewInMemory()
	defer func() { _ = pubSub.Close() }()

	// call register fn
	assert.NoError(t, srv.Register(context.Background(), config.Config{}, nil, stor, pubSub, pubSub))

	for _, r := range routes {
		route, _ := router.Get(r.name).GetPathTemplate()
		assert.Equal(t, r.route, route)
		methods, _ := router.Get(r.name).GetMethods()
		assert.Equal(t, r.methods, methods)
	}
}

package http

import (
	"fmt"
	"net/http"
	"testing"
	nullBroadcast "webhook-tester/broadcast/null"
	"webhook-tester/settings"
	nullStorage "webhook-tester/storage/null"

	"github.com/stretchr/testify/assert"
)

func TestServer_RegisterHandlers(t *testing.T) {
	t.Parallel()

	var (
		appSettings = &settings.AppSettings{}
		stor        = &nullStorage.Storage{}
		br          = &nullBroadcast.Broadcaster{}
	)

	var s = NewServer(&ServerSettings{}, appSettings, stor, br)

	var cases = []struct {
		giveName         string
		wantPathTemplate string
		wantMethods      []string
	}{
		{
			giveName:         "liveness_probe",
			wantPathTemplate: "/live",
			wantMethods:      []string{http.MethodGet},
		},
		{
			giveName:         "readiness_probe",
			wantPathTemplate: "/ready",
			wantMethods:      []string{http.MethodGet},
		},

		{
			giveName:         "api_settings_get",
			wantPathTemplate: "/api/settings",
			wantMethods:      []string{http.MethodGet},
		},
		{
			giveName:         "api_session_create",
			wantPathTemplate: "/api/session",
			wantMethods:      []string{http.MethodPost},
		},
		{
			giveName:         "api_session_delete",
			wantPathTemplate: "/api/session/{sessionUUID:" + uuidPattern + "}",
			wantMethods:      []string{http.MethodDelete},
		},
		{
			giveName:         "api_session_requests_all_get",
			wantPathTemplate: "/api/session/{sessionUUID:" + uuidPattern + "}/requests",
			wantMethods:      []string{http.MethodGet},
		},
		{
			giveName:         "api_session_request_get",
			wantPathTemplate: "/api/session/{sessionUUID:" + uuidPattern + "}/requests/{requestUUID:" + uuidPattern + "}",
			wantMethods:      []string{http.MethodGet},
		},
		{
			giveName:         "api_delete_session_request",
			wantPathTemplate: "/api/session/{sessionUUID:" + uuidPattern + "}/requests/{requestUUID:" + uuidPattern + "}",
			wantMethods:      []string{http.MethodDelete},
		},
		{
			giveName:         "api_delete_all_session_requests",
			wantPathTemplate: "/api/session/{sessionUUID:" + uuidPattern + "}/requests",
			wantMethods:      []string{http.MethodDelete},
		},

		{
			giveName:         "webhook",
			wantPathTemplate: "/{sessionUUID:" + uuidPattern + "}",
			wantMethods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodTrace,
			},
		},
		{
			giveName:         "webhook_with_status_code",
			wantPathTemplate: "/{sessionUUID:" + uuidPattern + "}/{statusCode:[1-5][0-9][0-9]}",
			wantMethods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodTrace,
			},
		},
		{
			giveName:         "webhook_any",
			wantPathTemplate: "/{sessionUUID:" + uuidPattern + "}/{any:.*}",
			wantMethods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodTrace,
			},
		},
	}

	for _, tt := range cases {
		assert.Nil(t,
			s.Router.Get(tt.giveName), fmt.Sprintf("Handler for route [%s] must be not registered", tt.giveName),
		)
	}

	s.RegisterHandlers()

	for _, tt := range cases {
		t.Run(tt.giveName, func(t *testing.T) {
			route := s.Router.Get(tt.giveName)

			pathTemplate, pathTemplateErr := route.GetPathTemplate()
			assert.Nil(t, pathTemplateErr)
			assert.Equal(t, tt.wantPathTemplate, pathTemplate)

			routeMethods, routeMethodsErr := route.GetMethods()
			assert.Nil(t, routeMethodsErr)
			assert.Equal(t, tt.wantMethods, routeMethods)
		})
	}
}

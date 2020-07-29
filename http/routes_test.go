package http

import (
	"fmt"
	"net/http"
	"testing"
	"webhook-tester/settings"

	"github.com/stretchr/testify/assert"
)

func TestServer_RegisterHandlers(t *testing.T) {
	t.Parallel()

	var s = NewServer(&ServerSettings{}, &settings.AppSettings{}, &fakeStorage{}, &fakeBroadcaster{})

	var cases = []struct {
		giveName         string
		wantPathTemplate string
		wantMethods      []string
	}{
		{
			giveName:         "ping",
			wantPathTemplate: "/ping",
			wantMethods:      []string{http.MethodGet},
		},
	}

	for _, tt := range cases {
		assert.Nil(t,
			s.Router.Get(tt.giveName),
			fmt.Sprintf("Handler for route [%s] must be not registered", tt.giveName),
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

package http

import (
	"testing"
	"time"
	appSettings "webhook-tester/settings"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	settings := ServerSettings{
		Address:          "1.2.3.4:321",
		WriteTimeout:     10 * time.Second,
		ReadTimeout:      13 * time.Second,
		KeepAliveEnabled: true,
	}

	server := NewServer(&settings, &appSettings.AppSettings{}, &fakeStorage{}, &fakeBroadcaster{})

	assert.Equal(t, &settings, server.settings)
	assert.Equal(t, "1.2.3.4:321", server.Server.Addr)
	assert.Equal(t, 10*time.Second, server.Server.WriteTimeout)
	assert.Equal(t, 13*time.Second, server.Server.ReadTimeout)
}

func TestServer_Start(t *testing.T) {
	t.Skip("Not implemented yet")
}

func TestServer_Stop(t *testing.T) {
	t.Skip("Not implemented yet")
}

package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "LISTEN_ADDR", string(ListenAddr))
	assert.Equal(t, "LISTEN_PORT", string(ListenPort))
	assert.Equal(t, "MAX_REQUESTS", string(MaxSessionRequests))
	assert.Equal(t, "SESSION_TTL", string(SessionTTL))
	assert.Equal(t, "STORAGE_DRIVER", string(StorageDriverName))
	assert.Equal(t, "PUBSUB_DRIVER", string(PubSubDriver))
	assert.Equal(t, "WS_MAX_CLIENTS", string(WebsocketMaxClients))
	assert.Equal(t, "WS_MAX_LIFETIME", string(WebsocketMaxLifetime))
	assert.Equal(t, "REDIS_DSN", string(RedisDSN))
	assert.Equal(t, "CREATE_SESSION", string(CreateSessionUUID))
}

func TestEnvVariable_Lookup(t *testing.T) {
	cases := []struct {
		giveEnv envVariable
	}{
		{giveEnv: ListenAddr},
		{giveEnv: ListenPort},
		{giveEnv: MaxSessionRequests},
		{giveEnv: SessionTTL},
		{giveEnv: StorageDriverName},
		{giveEnv: PubSubDriver},
		{giveEnv: WebsocketMaxClients},
		{giveEnv: WebsocketMaxLifetime},
		{giveEnv: RedisDSN},
		{giveEnv: CreateSessionUUID},
	}

	for _, tt := range cases {
		t.Run(tt.giveEnv.String(), func(t *testing.T) {
			assert.NoError(t, os.Unsetenv(tt.giveEnv.String())) // make sure that env is unset for test

			defer func() { assert.NoError(t, os.Unsetenv(tt.giveEnv.String())) }()

			value, exists := tt.giveEnv.Lookup()
			assert.False(t, exists)
			assert.Empty(t, value)

			assert.NoError(t, os.Setenv(tt.giveEnv.String(), "foo"))

			value, exists = tt.giveEnv.Lookup()
			assert.True(t, exists)
			assert.Equal(t, "foo", value)
		})
	}
}

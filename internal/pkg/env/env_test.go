package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "LISTEN_ADDR", string(ListenAddr))
	assert.Equal(t, "LISTEN_PORT", string(ListenPort))
	assert.Equal(t, "PUBLIC_DIR", string(PublicDir))
	assert.Equal(t, "MAX_REQUESTS", string(MaxSessionRequests))
	assert.Equal(t, "SESSION_TTL", string(SessionTTL))
	assert.Equal(t, "STORAGE_DRIVER", string(StorageDriverName))
	assert.Equal(t, "BROADCAST_DRIVER", string(BroadcastDriverName))
	assert.Equal(t, "PUSHER_APP_ID", string(PusherAppID))
	assert.Equal(t, "PUSHER_KEY", string(PusherKey))
	assert.Equal(t, "PUSHER_SECRET", string(PusherSecret))
	assert.Equal(t, "PUSHER_CLUSTER", string(PusherCluster))
	assert.Equal(t, "REDIS_DSN", string(RedisDSN))
}

func TestEnvVariable_Lookup(t *testing.T) {
	cases := []struct {
		giveEnv envVariable
	}{
		{giveEnv: ListenAddr},
		{giveEnv: ListenPort},
		{giveEnv: PublicDir},
		{giveEnv: MaxSessionRequests},
		{giveEnv: SessionTTL},
		{giveEnv: StorageDriverName},
		{giveEnv: BroadcastDriverName},
		{giveEnv: PusherAppID},
		{giveEnv: PusherSecret},
		{giveEnv: PusherCluster},
		{giveEnv: RedisDSN},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.giveEnv.String(), func(t *testing.T) {
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

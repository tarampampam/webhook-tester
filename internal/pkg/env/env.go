// Package env contains all about environment variables, that can be used by current application.
package env

import "os"

type envVariable string

const (
	// ListenAddr is IP address for listening.
	ListenAddr envVariable = "LISTEN_ADDR"

	// ListenPort is port number for listening.
	ListenPort envVariable = "LISTEN_PORT"

	// PublicDir is a directory with public resources.
	PublicDir envVariable = "PUBLIC_DIR"

	// MaxSessionRequests is a maximum stored requests per session.
	MaxSessionRequests envVariable = "MAX_REQUESTS"

	// SessionTTL is a session lifetime.
	SessionTTL envVariable = "SESSION_TTL"

	BroadcastDriverName envVariable = "BROADCAST_DRIVER"

	// PusherAppID is a pusher application ID.
	PusherAppID envVariable = "PUSHER_APP_ID"

	// PusherKey is a pusher key.
	PusherKey envVariable = "PUSHER_KEY"

	// PusherSecret is a pusher secret.
	PusherSecret envVariable = "PUSHER_SECRET"

	// PusherCluster is a pusher cluster.
	PusherCluster envVariable = "PUSHER_CLUSTER"

	// RedisDSN is URL-like redis connection string <https://redis.uptrace.dev/#connecting-to-redis-server>.
	RedisDSN envVariable = "REDIS_DSN"
)

// String returns environment variable name in the string representation.
func (e envVariable) String() string { return string(e) }

// Lookup retrieves the value of the environment variable. If the variable is present in the environment the value
// (which may be empty) is returned and the boolean is true. Otherwise the returned value will be empty and the
// boolean will be false.
func (e envVariable) Lookup() (string, bool) { return os.LookupEnv(string(e)) }

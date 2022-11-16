// Package env contains all about environment variables, that can be used by current application.
package env

import "os"

type envVariable string

const (
	ListenAddr           envVariable = "LISTEN_ADDR"     // IP address for listening
	ListenPort           envVariable = "LISTEN_PORT"     // port number for listening
	Port                 envVariable = "PORT"            // alias for ListenPort
	MaxSessionRequests   envVariable = "MAX_REQUESTS"    // maximum stored requests per session
	SessionTTL           envVariable = "SESSION_TTL"     // session lifetime
	StorageDriverName    envVariable = "STORAGE_DRIVER"  // storage driver name
	PubSubDriver         envVariable = "PUBSUB_DRIVER"   // pub/sub driver name
	WebsocketMaxClients  envVariable = "WS_MAX_CLIENTS"  // maximal websocket clients
	WebsocketMaxLifetime envVariable = "WS_MAX_LIFETIME" // maximal single websocket lifetime
	RedisDSN             envVariable = "REDIS_DSN"       // URL-like redis connection string <https://bit.ly/3maKq4l>
	CreateSessionUUID    envVariable = "CREATE_SESSION"  // Persistent session UUID, issue#160 <https://bit.ly/3Eyfupt>
)

// String returns environment variable name in the string representation.
func (e envVariable) String() string { return string(e) }

// Lookup retrieves the value of the environment variable. If the variable is present in the environment the value
// (which may be empty) is returned and the boolean is true. Otherwise, the returned value will be empty and the
// boolean will be false.
func (e envVariable) Lookup() (string, bool) { return os.LookupEnv(string(e)) }

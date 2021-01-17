package settings

import "time"

type AppSettings struct {
	// Maximum stored requests per session
	MaxRequests uint16

	// Session lifetime (TTL)
	SessionTTL time.Duration

	// Declared here HTTP header prefixes will be ignored for incoming webhook headers recording
	IgnoreHeaderPrefixes []string

	// pusher.com key
	PusherKey string

	// pusher.com cluster
	PusherCluster string
}

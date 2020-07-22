package settings

import "time"

type AppSettings struct {
	// Maximum stored requests per session
	MaxRequests uint16

	// Session lifetime (TTL)
	SessionTTL time.Duration
}

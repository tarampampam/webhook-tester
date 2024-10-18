package config

import "time"

type AppSettings struct {
	MaxRequests        uint16
	MaxRequestBodySize uint32
	SessionTTL         time.Duration
}

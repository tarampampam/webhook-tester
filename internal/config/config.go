package config

import (
	"net/url"
	"time"
)

type AppSettings struct {
	MaxRequests        uint16        // how many requests can be stored in the storage
	MaxRequestBodySize uint32        // max size of the request body
	SessionTTL         time.Duration // session time to live
	AutoCreateSessions bool          // feature: auto create sessions
	TunnelEnabled      bool          // feature: tunnel (public url to local server) enabled
	TunnelURL          *url.URL      // tunnel public url
	PublicURLRoot      *url.URL      // public URL root override for webhook URLs
}

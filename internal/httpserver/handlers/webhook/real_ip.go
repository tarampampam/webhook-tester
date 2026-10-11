package webhook

import (
	"net"
	"net/http"
	"strings"
)

// singleIPHeaders are CDN/proxy headers that contain exactly one IP.
// Order matters: most specific/trustworthy first.
var singleIPHeaders = []string{ //nolint:gochecknoglobals // package-level lookup table, immutable after init
	"CF-Connecting-IP",    // Cloudflare
	"True-Client-IP",      // Cloudflare Enterprise / Akamai
	"Fastly-Client-IP",    // Fastly
	"X-Real-IP",           // Nginx
	"X-Client-IP",         // some proxies
	"X-Cluster-Client-IP", // cluster load balancers
}

func extractRealIP(r *http.Request) string {
	// 1. CDN single-value headers - most reliable when present
	for _, h := range singleIPHeaders {
		if v := r.Header.Get(h); v != "" {
			if ip := net.ParseIP(strings.TrimSpace(v)); ip != nil {
				return ip.String()
			}
		}
	}

	// 2. X-Forwarded-For: take the leftmost valid IP (client-reported)
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for part := range strings.SplitSeq(v, ",") {
			if s := strings.TrimSpace(part); s != "" {
				if ip := net.ParseIP(s); ip != nil {
					return ip.String()
				}
			}
		}
	}

	// 3. fallback to RemoteAddr
	return remoteAddrIP(r)
}

// remoteAddrIP extracts the host from RemoteAddr, handling IPv6 correctly.
func remoteAddrIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

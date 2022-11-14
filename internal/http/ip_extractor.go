package http

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func NewIPExtractor() echo.IPExtractor {
	// we will trust following HTTP headers for the real ip extracting (priority low -> high).
	var trustHeaders = [...]string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}

	return func(r *http.Request) string {
		var ip string

		for _, name := range trustHeaders {
			if value := r.Header.Get(name); value != "" {
				// `X-Forwarded-For` can be `10.0.0.1, 10.0.0.2, 10.0.0.3`
				if strings.Contains(value, ",") {
					parts := strings.Split(value, ",")

					if len(parts) > 0 {
						ip = strings.TrimSpace(parts[0])
					}
				} else {
					ip = strings.TrimSpace(value)
				}
			}
		}

		if net.ParseIP(ip) != nil {
			return ip
		}

		return strings.Split(r.RemoteAddr, ":")[0]
	}
}

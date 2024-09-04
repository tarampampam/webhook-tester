package http_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	appHttp "gh.tarampamp.am/webhook-tester/internal/http"
)

func TestNewIPExtractor(t *testing.T) {
	var extractor = appHttp.NewIPExtractor()

	for name, tt := range map[string]struct {
		giveRequest func() *http.Request
		wantIP      string
	}{
		"IP from remote addr": {
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "4.3.2.1:567"

				return req
			},
			wantIP: "4.3.2.1",
		},
		"IP from 'CF-Connecting-IP' header": {
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "4.4.4.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
				req.Header.Set("X-Real-IP", "10.0.1.1")
				req.Header.Set("CF-Connecting-IP", "10.1.1.1")

				return req
			},
			wantIP: "10.1.1.1",
		},
		"IP from 'X-Real-IP' header": {
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "8.8.8.8:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
				req.Header.Set("X-Real-IP", "10.0.1.1")

				return req
			},
			wantIP: "10.0.1.1",
		},
		"IP from 'X-Forwarded-For' header": {
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "1.2.3.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")

				return req
			},
			wantIP: "10.0.0.1",
		},
	} {
		t.Run(name, func(t *testing.T) { assert.EqualValues(t, tt.wantIP, extractor(tt.giveRequest())) })
	}
}

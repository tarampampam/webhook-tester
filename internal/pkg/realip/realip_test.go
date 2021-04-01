package realip_test

import (
	"net/http"
	"testing"

	"github.com/tarampampam/webhook-tester/internal/pkg/realip"
)

func TestFromHTTPRequest(t *testing.T) {
	for _, tt := range []struct {
		name        string
		giveRequest func() *http.Request
		wantIP      string
	}{
		{
			name: "IP from remote addr",
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "4.3.2.1:567"

				return req
			},
			wantIP: "4.3.2.1",
		},
		{
			name: "IP from 'CF-Connecting-IP' header",
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
		{
			name: "IP from 'X-Real-IP' header",
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "8.8.8.8:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
				req.Header.Set("X-Real-IP", "10.0.1.1")

				return req
			},
			wantIP: "10.0.1.1",
		},
		{
			name: "IP from 'X-Forwarded-For' header",
			giveRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "1.2.3.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")

				return req
			},
			wantIP: "10.0.0.1",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := realip.FromHTTPRequest(tt.giveRequest()); got != tt.wantIP {
				t.Errorf("want IP: %s, got: %s", tt.wantIP, got)
			}
		})
	}
}

func BenchmarkFromHTTPRequest(b *testing.B) {
	b.ReportAllocs()

	req, _ := http.NewRequest(http.MethodGet, "http://testing", nil)
	req.RemoteAddr = "4.4.4.4:567"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	req.Header.Set("X-Real-IP", "10.0.1.1")
	req.Header.Set("CF-Connecting-IP", "10.1.1.1")

	for i := 0; i < b.N; i++ {
		if realip.FromHTTPRequest(req) != "10.1.1.1" {
			b.Fail()
		}
	}
}

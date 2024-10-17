package healthcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HealthChecker is a heals checker.
type HealthChecker struct {
	httpClient httpClient
}

const (
	defaultHTTPClientTimeout = time.Second * 3

	UserAgent = "HealthChecker/webhook-tester"
	Route     = "/healthz"
	Method    = http.MethodGet
)

// NewHealthChecker creates heals checker.
func NewHealthChecker(client ...httpClient) *HealthChecker {
	var c httpClient

	if len(client) == 1 {
		c = client[0]
	} else {
		c = &http.Client{ // default
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
			Timeout: defaultHTTPClientTimeout,
		}
	}

	return &HealthChecker{httpClient: c}
}

// Check application using liveness probe.
func (c *HealthChecker) Check(ctx context.Context, httpPort uint) error {
	var uri = fmt.Sprintf("http://127.0.0.1:%d%s", httpPort, Route)

	req, err := http.NewRequestWithContext(ctx, Method, uri, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	_ = resp.Body.Close()

	if code := resp.StatusCode; code != http.StatusOK {
		return fmt.Errorf("wrong status code [%d] from the live endpoint (%s)", code, uri)
	}

	return nil
}

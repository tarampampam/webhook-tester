package ft

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/cmd/webhook-tester/app"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// RunApp starts the application in a separate goroutine with the given arguments, waits until it is ready to
// accept requests, and returns the base URL and a function to stop the app.
//
// NOTE: Do not use `--port` in the arguments, as RunApp will automatically acquire a free port and pass it to
// the app.
func RunApp(t *testing.T, args ...string) (string /* base URL */, error) {
	t.Helper()

	const (
		pollInterval   = 10 * time.Millisecond
		startupTimeout = 10 * time.Second
	)

	// determine the schema based on the presence of TLS-related flags in the arguments
	schema := "http"

	for _, arg := range args {
		if strings.Contains(arg, "-https-key-file") ||
			strings.Contains(arg, "-https-cert-file") ||
			strings.Contains(arg, "-self-signed-tls") {
			schema = "https"

			break
		}
	}

	// acquire a free port by briefly listening on
	ln, lnErr := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	assert.NoError(t, lnErr)

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("expected *net.TCPAddr from tcp listener") // will never happen
	}

	appCtx, stopApp := context.WithCancel(t.Context())
	t.Cleanup(stopApp)

	// this channel is used to signal when the app has exited, and to capture any error it returns
	done := make(chan error, 1)

	// run the app in a separate goroutine (it should never exit until the context is canceled, as it is an HTTP server)
	go func(ctx context.Context) {
		defer close(done)

		// close the listener to free the port for the app
		_ = ln.Close()

		// NOTE: between port closing and the app starting to listen, there is a small TOCTOU window where another
		// process can bind the port
		done <- app.NewApp("webhook-tester-unit-test").Run(
			ctx,
			append([]string{
				"--port", strconv.Itoa(addr.Port),
				"--log-level", "error", // reduce noise in test output
			}, args...),
		)
	}(appCtx)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	pollCtx, cancelPoll := context.WithTimeout(t.Context(), startupTimeout)
	defer cancelPoll()

	baseUrl := schema + "://" + addr.String()
	readinessURL := baseUrl + "/ready"

	for {
		// if the app is not ready yet - wait for the next tick, or for the app to exit
		select {
		case <-ticker.C: // tick has come
			req, reqErr := http.NewRequestWithContext(pollCtx, http.MethodGet, readinessURL, http.NoBody)
			assert.NoError(t, reqErr)

			// check the app's readiness endpoint
			resp, respErr := client.Do(req)
			if respErr == nil {
				_ = resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					return baseUrl, nil
				}
			}
		case err := <-done: // app's goroutine has exited
			cancelPoll()

			return "", fmt.Errorf("app exited before becoming ready: %w", err)
		case <-pollCtx.Done():
			return "", fmt.Errorf("app did not become ready within %s: %w", startupTimeout, pollCtx.Err())
		}
	}
}

// MustRunApp is a helper that calls RunApp and fails the test if it returns an error.
func MustRunApp(t *testing.T, args ...string) string {
	t.Helper()

	baseUrl, err := RunApp(t, args...)
	assert.NoError(t, err)

	return baseUrl
}

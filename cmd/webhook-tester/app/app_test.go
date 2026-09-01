package app_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/cmd/webhook-tester/app"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// freePort briefly listens on 127.0.0.1:0 to obtain a free TCP port, then closes it.
// There is a small TOCTOU window between close and the caller binding the port, which is
// acceptable in test scenarios.
func freePort(t *testing.T) uint {
	t.Helper()

	ln, lnErr := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, lnErr)

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("expected *net.TCPAddr from tcp listener")
	}

	assert.NoError(t, ln.Close())

	return uint(tcpAddr.Port) //nolint:gosec // port is always in [1, 65535]
}

// pollReady polls GET <url> every 10 ms until it receives HTTP 200 or the deadline (10 s) expires.
func pollReady(t *testing.T, client *http.Client, url string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second) //nolint:mnd // 10 s is a reasonable test timeout

	for time.Now().Before(deadline) {
		req, reqErr := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if reqErr != nil {
			t.Fatalf("build readiness request: %v", reqErr)
		}

		resp, doErr := client.Do(req)
		if doErr == nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(10 * time.Millisecond) //nolint:mnd // polling interval
	}

	t.Fatalf("server at %s did not become ready within 10s", url)
}

func TestApp_Run_HappyPath(t *testing.T) {
	t.Parallel()

	port := freePort(t)

	sessionsFile := filepath.Join(t.TempDir(), "sessions.json")
	assert.NoError(t, os.WriteFile(sessionsFile, []byte(`{
		"sessions": [
			{"id": "123e4567-e89b-12d3-a456-426614174000", "code": 201, "body": "hello"}
		]
	}`), 0o600))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- app.NewApp("webhook-tester").Run(ctx, []string{
			"--log-level=debug",
			"--log-format=json",
			"--addr=127.0.0.1",
			"--port=" + strconv.Itoa(int(port)),
			"--http-read-header-timeout=1s",
			"--http-read-timeout=5s",
			"--http-idle-timeout=10s",
			"--http-shutdown-timeout=100ms",
			"--pubsub-driver=memory",
			"--storage-driver=memory",
			"--max-requests=10",
			"--session-ttl=1h",
			"--max-request-body-size=1024",
			"--auto-create-sessions",
			"--public-url-root=http://localhost",
			"--sessions-file=" + sessionsFile,
		})
	}()

	pollReady(t, &http.Client{}, fmt.Sprintf("http://127.0.0.1:%d/ready", port))

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second): //nolint:mnd
		t.Fatal("app did not stop within 10 seconds after context cancellation")
	}
}

func TestApp_Run_SelfSignedTLS(t *testing.T) {
	t.Parallel()

	port := freePort(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- app.NewApp("webhook-tester").Run(ctx, []string{
			"--addr=127.0.0.1",
			"--port=" + strconv.Itoa(int(port)),
			"--http-shutdown-timeout=100ms",
			"--self-signed-tls",
		})
	}()

	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // intentional for self-signed cert
		},
	}

	pollReady(t, tlsClient, fmt.Sprintf("https://127.0.0.1:%d/ready", port))

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second): //nolint:mnd
		t.Fatal("app did not stop within 10 seconds after context cancellation")
	}
}

func TestApp_Run_InvalidFlags(t *testing.T) { //nolint:funlen // table-driven test with many flag validation cases
	t.Parallel()

	for name, tc := range map[string]struct {
		giveArgs        []string
		wantErrContains string
	}{
		"invalid log level": {
			giveArgs:        []string{"--log-level=garbage"},
			wantErrContains: "unrecognized logging level",
		},
		"invalid log format": {
			giveArgs:        []string{"--log-format=garbage"},
			wantErrContains: "unrecognized logging format",
		},
		"invalid addr": {
			giveArgs:        []string{"--addr=not-an-ip"},
			wantErrContains: "wrong IP address",
		},
		"port zero": {
			giveArgs:        []string{"--port=0"},
			wantErrContains: "wrong TCP port number",
		},
		"port too large": {
			giveArgs:        []string{"--port=99999"},
			wantErrContains: "wrong TCP port number",
		},
		"negative read header timeout": {
			giveArgs:        []string{"--http-read-header-timeout=-1s"},
			wantErrContains: "http read header timeout must be a non-negative duration",
		},
		"negative read timeout": {
			giveArgs:        []string{"--http-read-timeout=-1s"},
			wantErrContains: "http read timeout must be a non-negative duration",
		},
		"negative idle timeout": {
			giveArgs:        []string{"--http-idle-timeout=-1s"},
			wantErrContains: "http idle timeout must be a non-negative duration",
		},
		"negative shutdown timeout": {
			giveArgs:        []string{"--http-shutdown-timeout=-1s"},
			wantErrContains: "http shutdown timeout must be a non-negative duration",
		},
		"invalid redis dsn": {
			giveArgs:        []string{"--redis-dsn=not-a-dsn"},
			wantErrContains: "invalid Redis DSN",
		},
		"unknown pubsub driver": {
			giveArgs:        []string{"--pubsub-driver=garbage"},
			wantErrContains: "unknown Pub/Sub driver name",
		},
		"unknown storage driver": {
			giveArgs:        []string{"--storage-driver=garbage"},
			wantErrContains: "unknown storage driver name",
		},
		"negative session ttl": {
			giveArgs:        []string{"--session-ttl=-1s"},
			wantErrContains: "session TTL must be a positive duration",
		},
		"max request body size too large": {
			giveArgs:        []string{"--max-request-body-size=99999"},
			wantErrContains: "max request body size",
		},
		"max requests too large": {
			giveArgs:        []string{"--max-requests=99999"},
			wantErrContains: "max requests value",
		},
		"public url root wrong scheme": {
			giveArgs:        []string{"--public-url-root=ftp://example.com"},
			wantErrContains: "scheme must be http or https",
		},
		"tunnel url wrong scheme": {
			giveArgs:        []string{"--tunnel-url=ftp://example.com"},
			wantErrContains: "scheme must be http or https",
		},
		"unknown tunnel driver": {
			giveArgs:        []string{"--tunnel-driver=garbage"},
			wantErrContains: "unknown tunnel driver name",
		},
		"nonexistent cert file": {
			giveArgs:        []string{"--https-cert-file=/nonexistent/cert.pem"},
			wantErrContains: "does not exist",
		},
		"nonexistent key file": {
			giveArgs:        []string{"--https-key-file=/nonexistent/key.pem"},
			wantErrContains: "does not exist",
		},
		"nonexistent fs storage dir": {
			giveArgs:        []string{"--fs-storage-dir=/nonexistent/storage"},
			wantErrContains: "does not exist",
		},
		"nonexistent sessions file": {
			giveArgs:        []string{"--sessions-file=/nonexistent/sessions.json"},
			wantErrContains: "does not exist",
		},
		"empty addr": {
			giveArgs:        []string{"--addr="},
			wantErrContains: "missing IP address",
		},
		"public url root missing host": {
			giveArgs:        []string{"--public-url-root=http://"},
			wantErrContains: "missing host",
		},
		"tunnel url missing host": {
			giveArgs:        []string{"--tunnel-url=http://"},
			wantErrContains: "missing host",
		},
		"fs storage driver without dir": {
			giveArgs:        []string{"--storage-driver=fs"},
			wantErrContains: "http server failed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := app.NewApp("webhook-tester").Run(t.Context(), tc.giveArgs)
			assert.ErrorContains(t, err, tc.wantErrContains)
		})
	}
}

func TestApp_Help(t *testing.T) {
	t.Parallel()

	assert.Contains(t, app.NewApp("webhook-tester").Help(), "webhook-tester")
}

// TestApp_Run_InvalidFlags_PathValidation covers the IsDir/IsNotDir branches in path flag validators
// that require actual filesystem objects and cannot be expressed as plain string args.
func TestApp_Run_InvalidFlags_PathValidation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tmpFile, tmpErr := os.CreateTemp(dir, "*.pem")
	assert.NoError(t, tmpErr)
	assert.NoError(t, tmpFile.Close())

	for name, tc := range map[string]struct {
		giveArgs        []string
		wantErrContains string
	}{
		"cert file is a directory": {
			giveArgs:        []string{"--https-cert-file=" + dir},
			wantErrContains: "is a directory",
		},
		"key file is a directory": {
			giveArgs:        []string{"--https-key-file=" + dir},
			wantErrContains: "is a directory",
		},
		"fs storage path is a file": {
			giveArgs:        []string{"--fs-storage-dir=" + tmpFile.Name()},
			wantErrContains: "is not a directory",
		},
		"sessions file is a directory": {
			giveArgs:        []string{"--sessions-file=" + dir},
			wantErrContains: "is a directory",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := app.NewApp("webhook-tester").Run(t.Context(), tc.giveArgs)
			assert.ErrorContains(t, err, tc.wantErrContains)
		})
	}
}

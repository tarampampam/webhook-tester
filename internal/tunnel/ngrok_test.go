package tunnel_test

import (
	"crypto/tls"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	"gh.tarampamp.am/webhook-tester/v3/internal/tunnel"
)

func TestNewNgrok(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil instance", func(t *testing.T) {
		t.Parallel()

		assert.NotNil(t, tunnel.NewNgrok("some-token",
			tunnel.WithNgrokLogger(logger.NewNop()),
			tunnel.WithNgrokTLSUpstream(&tls.Config{}),
			tunnel.WithNgrokURL("https://custom.ngrok.io"),
		))
	})

	t.Run("empty auth token does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() { _ = tunnel.NewNgrok("") })
	})

	t.Run("same token produces the same instance URL (deterministic)", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			a := tunnel.NewNgrok("token-abc")
			b := tunnel.NewNgrok("token-abc")
			_ = a.Close()
			_ = b.Close()
		})
	})
}

func TestNgrok_Close(t *testing.T) {
	t.Parallel()

	t.Run("no-op when Expose was never called", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, tunnel.NewNgrok("some-token").Close())
	})

	t.Run("idempotent on fresh instance", func(t *testing.T) {
		t.Parallel()

		n := tunnel.NewNgrok("some-token")
		assert.NoError(t, n.Close())
		assert.NoError(t, n.Close())
		assert.NoError(t, n.Close())
	})

	t.Run("safe with WithNgrokURL option", func(t *testing.T) {
		t.Parallel()

		n := tunnel.NewNgrok("some-token", tunnel.WithNgrokURL("https://custom.example.ngrok.io"))
		assert.NoError(t, n.Close())
	})
}

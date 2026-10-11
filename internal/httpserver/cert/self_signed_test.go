package cert_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"path/filepath"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/cert"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestSelfSignedTLS(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir() // empty dir

	got, err := cert.SelfSignedTLS(tmpDir)
	assert.NoError(t, err)

	priv, ok := got.PrivateKey.(*ecdsa.PrivateKey)
	assert.True(t, ok)
	assert.Equal(t, elliptic.P256(), priv.Curve)
	assert.True(t, len(got.Certificate) > 0)

	assert.FileExists(t, filepath.Join(tmpDir, "webhook-tester-self-signed.crt"))
	assert.FileExists(t, filepath.Join(tmpDir, "webhook-tester-self-signed.key"))

	got2, err := cert.SelfSignedTLS(tmpDir) // should load existing now
	assert.NoError(t, err)

	assert.DeepEqual(t, got, got2)
}

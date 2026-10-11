package http2https_test

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/http2https"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// newTestCert generates a minimal self-signed certificate for use in tests.
func newTestCert(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), //nolint:mnd
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	derBytes, derErr := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	assert.NoError(t, derErr)

	keyDER, keyErr := x509.MarshalPKCS8PrivateKey(key)
	assert.NoError(t, keyErr)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	cert, certErr := tls.X509KeyPair(certPEM, keyPEM)
	assert.NoError(t, certErr)

	return cert
}

func TestNewListener(t *testing.T) {
	t.Parallel()

	ln, lnErr := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, lnErr)

	wrapped := http2https.NewListener(ln)

	t.Cleanup(func() { _ = wrapped.Close() })
	assert.NotNil(t, wrapped)
	assert.Equal(t, ln.Addr().String(), wrapped.Addr().String())
	assert.Equal(t, ln, wrapped.Unwrap())
	assert.NoError(t, wrapped.Close())
}

func TestListener_PlainHTTPRedirect(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		request      string
		wantLocation string
	}{
		"root path": {
			request:      "GET / HTTP/1.1\r\nHost: example.com:8443\r\n\r\n",
			wantLocation: "https://example.com:8443/",
		},
		"path with query": {
			request:      "GET /foo?bar=baz HTTP/1.1\r\nHost: webhook.test\r\n\r\n",
			wantLocation: "https://webhook.test/foo?bar=baz",
		},
		"POST preserves method": {
			request:      "POST /hook HTTP/1.1\r\nHost: test.local\r\nContent-Length: 0\r\n\r\n",
			wantLocation: "https://test.local/hook",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ln, lnErr := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, lnErr)

			wrapped := http2https.NewListener(ln)
			defer func() { _ = wrapped.Close() }()

			// run accept loop; plain HTTP conns are handled internally and never returned here
			go func() {
				for {
					conn, err := wrapped.Accept()
					if err != nil {
						return
					}

					_ = conn.Close()
				}
			}()

			conn, dialErr := net.Dial("tcp", ln.Addr().String())
			assert.NoError(t, dialErr)

			defer func() { _ = conn.Close() }()

			assert.NoError(t, conn.SetDeadline(time.Now().Add(3*time.Second))) //nolint:mnd

			_, writeErr := conn.Write([]byte(tc.request))
			assert.NoError(t, writeErr)

			resp, readErr := http.ReadResponse(bufio.NewReader(conn), nil)
			assert.NoError(t, readErr)

			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
			assert.Equal(t, tc.wantLocation, resp.Header.Get("Location"))
			assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		})
	}
}

func TestListener_TLSConnection(t *testing.T) {
	t.Parallel()

	cert := newTestCert(t)

	ln, lnErr := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, lnErr)

	// http2https detects and redirects plain HTTP; TLS wrapping is applied on top by the caller
	wrapped := http2https.NewListener(ln)
	tlsLn := tls.NewListener(wrapped, &tls.Config{Certificates: []tls.Certificate{cert}})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second} //nolint:mnd

	go func() { _ = srv.Serve(tlsLn) }()

	defer func() { _ = srv.Close() }()

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}}

	resp, httpErr := client.Get("https://" + ln.Addr().String() + "/")
	assert.NoError(t, httpErr)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

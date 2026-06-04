package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const selfSignedCertFileName = "webhook-tester-self-signed" // without extension

// SelfSignedTLS returns a [tls.Certificate], loading it from disk if possible or generating a new self‑signed
// certificate pair if not. When a new pair is created and cacheDirPath points to an existing directory,
// the certificate and key are written to that directory.
//
// The resulting certificate is valid for localhost access (127.0.0.1, ::1, and the current hostname).
//
// All the errors related to loading or saving the certificate are ignored (and this is conscious) due to they are
// not critical.
func SelfSignedTLS(cacheDirPath string) (*tls.Certificate, error) {
	const (
		certFileName = selfSignedCertFileName + ".crt"
		keyFileName  = selfSignedCertFileName + ".key"
	)

	var (
		certPath, keyPath string // absolute file paths (if cacheDirPath is valid)
		cert, key         []byte // PEM-encoded data we either load or generate
	)

	// if a cache directory is provided and exists, precompute the file paths where we will attempt to load
	// from (or later save to)
	if cacheDirPath != "" {
		if stat, err := os.Stat(cacheDirPath); err == nil && stat.IsDir() {
			certPath = filepath.Join(cacheDirPath, certFileName)
			keyPath = filepath.Join(cacheDirPath, keyFileName)
		}
	}

	// happy path - try to load an existing certificate and key from disk
	if certPath != "" && keyPath != "" {
		if data, err := os.ReadFile(certPath); err == nil {
			cert = data
		}

		if data, err := os.ReadFile(keyPath); err == nil {
			key = data
		}

		// if both files were readable, attempt to parse them as a key pair
		if len(cert) > 0 && len(key) > 0 {
			if pair, err := tls.X509KeyPair(cert, key); err == nil {
				return &pair, nil
			}
		}
	}

	// otherwise, generate a new self-signed certificate
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa key: %w", err)
	}

	pub := &priv.PublicKey

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128)) //nolint:mnd
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	hostname, _ := os.Hostname() //nolint:errcheck

	// prepare the x509 certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "webhook-tester self-signed",
			Organization: []string{"webhook-tester"},
		},
		NotBefore:             now.Add(-24 * time.Hour),           // valid starting a day ago
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour), // roughly 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,

		DNSNames:    []string{"localhost", hostname},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// self-sign the certificate using the template as both parent and child
	derCert, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	// marshal the private key in PKCS#8 for maximum compatibility
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	// encode cert and key into PEM for tls.X509KeyPair
	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCert})
	key = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	// build a tls.Certificate from the freshly created PEM blobs
	pair, pairErr := tls.X509KeyPair(cert, key)
	if pairErr != nil {
		return nil, fmt.Errorf("create tls key pair: %w", pairErr)
	}

	// try to save the generated certificate and key
	if certPath != "" && keyPath != "" {
		// errors here are non-fatal, we just skip saving if something goes wrong
		_ = os.WriteFile(certPath, cert, 0o644) //nolint:mnd,gosec,errcheck // public cert can be world-readable
		_ = os.WriteFile(keyPath, key, 0o600)   //nolint:mnd,errcheck // private key should be owner-only
	}

	return &pair, nil
}

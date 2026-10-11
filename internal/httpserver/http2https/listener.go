// Package http2https provides a [net.Listener] wrapper that detects plain HTTP connections on a port
// intended for HTTPS and responds with an HTTP 307 redirect to the HTTPS equivalent URL. All other
// connections (e.g. TLS) are returned to the caller unchanged.
package http2https

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

const (
	// tlsClientHello is the first byte of a TLS record, which identifies the connection as TLS.
	tlsClientHello byte = 0x16

	// redirectDeadline is the maximum time allowed to read a plain HTTP request and write the redirect response.
	redirectDeadline = 5 * time.Second
)

// Listener wraps a [net.Listener] and intercepts plain HTTP connections on a port intended for HTTPS.
// Plain HTTP connections receive a 307 redirect to the HTTPS equivalent URL and are closed without being
// returned from Accept. All other connections (TLS or otherwise) are returned to the caller as-is; the
// caller is responsible for applying TLS (e.g. via [crypto/tls.NewListener]).
type Listener struct {
	inner net.Listener
}

var ( // compile-time interface assertions
	_ net.Listener                       = (*Listener)(nil)
	_ interface{ Unwrap() net.Listener } = (*Listener)(nil)
)

// NewListener wraps ln so that plain HTTP connections receive a 307 redirect to the HTTPS equivalent URL
// and are closed without being returned from Accept. All other connections are returned unchanged.
func NewListener(ln net.Listener) *Listener { return &Listener{inner: ln} }

// Accept waits for and returns the next non-HTTP connection. Plain HTTP connections are handled
// internally: a 307 redirect to the HTTPS equivalent URL is sent and the connection is closed.
// Accept only returns when a non-HTTP connection arrives, or when the underlying listener returns an error.
func (l *Listener) Accept() (net.Conn, error) {
	for {
		raw, err := l.inner.Accept()
		if err != nil {
			return nil, err
		}

		// use a minimal bufio reader to peek at the first byte without consuming it from the underlying conn
		br := bufio.NewReaderSize(raw, 1)

		first, peekErr := br.Peek(1)
		if peekErr != nil {
			_ = raw.Close()

			continue
		}

		if first[0] != tlsClientHello {
			// br is passed directly to avoid wrapping it in a second bufio.Reader inside sendRedirect
			go sendRedirect(raw, br)

			continue
		}

		// peekedConn replays the bytes already buffered by br so the caller (e.g. tls.NewListener) sees
		// the full ClientHello when performing the TLS handshake
		return &peekedConn{Conn: raw, r: br}, nil
	}
}

// Close closes the underlying listener.
func (l *Listener) Close() error { return l.inner.Close() }

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr { return l.inner.Addr() }

// Unwrap returns the underlying [net.Listener], allowing callers to access the original listener
// (e.g. to retrieve the real address or file descriptor after wrapping).
func (l *Listener) Unwrap() net.Listener { return l.inner }

// SetDeadline sets the deadline on the underlying listener if it supports the operation.
// Returns [errors.ErrUnsupported] if the inner listener does not implement SetDeadline.
func (l *Listener) SetDeadline(t time.Time) error {
	if sd, ok := l.inner.(interface{ SetDeadline(time.Time) error }); ok {
		return sd.SetDeadline(t)
	}

	return errors.ErrUnsupported
}

// peekedConn is a [net.Conn] that drains the bytes already buffered by a [bufio.Reader] before reading
// from the underlying connection. Required because [bufio.Reader.Peek] may buffer more bytes than requested.
//
// [syscall.Conn] is proxied to the underlying connection so that callers retain access to socket-level
// operations (e.g. [net.TCPConn] socket options needed by HTTP/2 through [crypto/tls.Conn.SyscallConn]).
//
// [io.ReaderFrom] is intentionally NOT implemented: when this conn is wrapped by a TLS layer (e.g.
// [crypto/tls.NewListener]), exposing ReadFrom would allow writes to bypass TLS encryption.
type peekedConn struct {
	net.Conn

	r *bufio.Reader
}

var ( // compile-time interface assertions
	_ net.Conn     = (*peekedConn)(nil)
	_ syscall.Conn = (*peekedConn)(nil)
)

// Read reads from the buffered reader first (replaying peeked bytes), then falls through to the underlying connection.
func (c *peekedConn) Read(b []byte) (int, error) { return c.r.Read(b) }

// SyscallConn proxies to the underlying connection's [syscall.Conn] if available, enabling socket-level access
// used by [crypto/tls.Conn.SyscallConn] to configure TCP options (e.g. TCP_NODELAY for HTTP/2).
func (c *peekedConn) SyscallConn() (syscall.RawConn, error) {
	if sc, ok := c.Conn.(syscall.Conn); ok {
		return sc.SyscallConn()
	}

	return nil, errors.ErrUnsupported
}

//nolint:gochecknoglobals // package-level replacer avoids per-call allocation
var crlfReplacer = strings.NewReplacer("\r", "", "\n", "")

// sendRedirect reads the HTTP request from conn using the already-buffered reader r, then responds with a
// 307 redirect to the HTTPS equivalent URL. The connection is always closed when this function returns.
func sendRedirect(conn net.Conn, r *bufio.Reader) {
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(redirectDeadline)); err != nil {
		return
	}

	req, err := http.ReadRequest(r)
	if err != nil {
		return
	}

	host := req.Host
	if host == "" {
		// fall back to the listener's port with localhost when no Host header is present (HTTP/1.0)
		_, port, splitErr := net.SplitHostPort(conn.LocalAddr().String())
		if splitErr != nil {
			return
		}

		host = net.JoinHostPort("localhost", port)
	}

	// strip CR/LF to prevent response splitting if the HTTP parser passes them through
	host = crlfReplacer.Replace(host)
	requestURI := crlfReplacer.Replace(req.RequestURI)

	if _, writeErr := fmt.Fprintf(conn, "HTTP/1.1 307 Temporary Redirect\r\n"+
		"Location: https://%s%s\r\n"+
		"Content-Length: 0\r\n"+
		"Cache-Control: no-store\r\n"+
		"X-Hint: This redirect was generated by Webhook Tester because you made a plain HTTP request to an HTTPS port\r\n"+
		"Connection: close\r\n\r\n", host, requestURI); writeErr != nil {
		return
	}
}

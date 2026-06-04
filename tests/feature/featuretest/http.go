package ft

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// client is a shared HTTP client with TLS verification disabled, used for testing HTTP/HTTPS endpoints with
// self-signed certificates.
var client = http.Client{ //nolint:gochecknoglobals
	Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec // testing only
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// Get makes an HTTP GET request to the specified URL.
// Returned response body is re-readable, allowing multiple reads.
func Get(t *testing.T, url string) (*http.Response, error) {
	t.Helper()

	r, rErr := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	assert.NoError(t, rErr)

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	makeResponseBodyReReadable(t, resp)

	return resp, nil
}

// MustGet makes an HTTP GET request and fails the test if any error occurs.
func MustGet(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := Get(t, url)
	assert.NoError(t, err)

	return resp
}

// Post makes an HTTP POST request to the specified URL with the given headers and body.
// Returned response body is re-readable, allowing multiple reads.
func Post(t *testing.T, url string, headers map[string]string, body []byte) (*http.Response, error) {
	t.Helper()

	r, rErr := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(body))
	assert.NoError(t, rErr)

	for k, v := range headers {
		r.Header.Set(k, v)
	}

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	makeResponseBodyReReadable(t, resp)

	return resp, nil
}

// MustPost makes an HTTP POST request and fails the test if any error occurs.
func MustPost(t *testing.T, url string, headers map[string]string, body []byte) *http.Response {
	t.Helper()

	resp, err := Post(t, url, headers, body)
	assert.NoError(t, err)

	return resp
}

// Request makes an HTTP request with the given method, URL, optional headers, and optional body.
// Returned response body is re-readable, allowing multiple reads.
func Request(t *testing.T, method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	t.Helper()

	var bodyReader io.Reader = http.NoBody
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	r, rErr := http.NewRequestWithContext(t.Context(), method, url, bodyReader)
	assert.NoError(t, rErr)

	for k, v := range headers {
		r.Header.Set(k, v)
	}

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	makeResponseBodyReReadable(t, resp)

	return resp, nil
}

// MustRequest makes an HTTP request and fails the test if any error occurs.
func MustRequest(t *testing.T, method, url string, headers map[string]string, body []byte) *http.Response {
	t.Helper()

	resp, err := Request(t, method, url, headers, body)
	assert.NoError(t, err)

	return resp
}

// Delete makes an HTTP DELETE request to the specified URL.
// Returned response body is re-readable, allowing multiple reads.
func Delete(t *testing.T, url string) (*http.Response, error) {
	t.Helper()

	return Request(t, http.MethodDelete, url, nil, nil)
}

// MustDelete makes an HTTP DELETE request and fails the test if any error occurs.
func MustDelete(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := Delete(t, url)
	assert.NoError(t, err)

	return resp
}

// ReadBody reads the entire response body and returns it as a byte slice.
func ReadBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	if v, ok := resp.Body.(io.Seeker); ok {
		_, sErr := v.Seek(0, io.SeekStart) // reset the body reader for potential future reads
		assert.NoError(t, sErr)
	}

	return body
}

// --------------------------------------------------------------------------------------------------------------------

// reReadableBody wraps a [bytes.Reader] to implement [io.ReadSeekCloser], allowing the response body to be read
// multiple times without re-allocating.
type reReadableBody struct{ *bytes.Reader }

func (reReadableBody) Close() error { return nil }

// makeResponseBodyReReadable replaces the response body with a re-readable version, allowing it to be read
// multiple times.
func makeResponseBodyReReadable(t *testing.T, resp *http.Response) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.NoError(t, err)

	resp.Body = reReadableBody{bytes.NewReader(body)}
}

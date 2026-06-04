package loader

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

// LoadSessions reads user-defined sessions from the specified JSON file and returns a map of session ID to its
// configured response.
func LoadSessions(path string) (map[string]storage.SessionResponse, error) {
	type session struct {
		ID      string            `json:"id"`
		Code    uint16            `json:"code"`
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
		Delay   sessionDuration   `json:"delay"`
	}

	type sessionFile struct {
		Sessions []session `json:"sessions"`
	}

	f, openErr := os.Open(path)
	if openErr != nil {
		return nil, fmt.Errorf("read sessions file: %w", openErr)
	}

	defer func() { _ = f.Close() }()

	var sf sessionFile
	if err := json.NewDecoder(f).Decode(&sf); err != nil {
		return nil, fmt.Errorf("parse sessions file: %w", err)
	}

	if err := f.Close(); err != nil {
		return nil, err
	}

	result := make(map[string]storage.SessionResponse, len(sf.Sessions))

	for i, s := range sf.Sessions {
		if s.ID == "" {
			return nil, fmt.Errorf("session at index %d: missing required field \"id\"", i)
		}

		body, bodyErr := parseSessionBody(s.Body)
		if bodyErr != nil {
			return nil, fmt.Errorf("session %q body: %w", s.ID, bodyErr)
		}

		headers := make([]storage.ResponseHeader, 0, len(s.Headers))
		for name, value := range s.Headers {
			headers = append(headers, storage.ResponseHeader{Name: name, Value: value})
		}

		result[s.ID] = storage.SessionResponse{
			Code:    s.Code,
			Headers: headers,
			Body:    body,
			Delay:   s.Delay.Duration,
		}
	}

	return result, nil
}

// parseSessionBody converts the raw JSON body value into a byte slice.
// Supported JSON types:
//   - string: first tries base64 decoding; falls back to the raw string bytes
//   - array of numbers: each element must be an integer in [0, 255]
//   - null / absent: returns nil
func parseSessionBody(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	// try string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if decoded, decErr := base64.StdEncoding.DecodeString(s); decErr == nil {
			return decoded, nil
		}

		return []byte(s), nil
	}

	// try array
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		buf := make([]byte, len(arr))

		for i, elem := range arr {
			var n float64
			if err = json.Unmarshal(elem, &n); err != nil {
				return nil, fmt.Errorf("element %d is not a number", i)
			}

			if n < 0 || n > math.MaxUint8 || math.Trunc(n) != n {
				return nil, fmt.Errorf("element %d value %v is out of byte range [0, 255]", i, n)
			}

			buf[i] = byte(n)
		}

		return buf, nil
	}

	return nil, errors.New("unsupported type: must be a string or an array of bytes")
}

// sessionDuration is a custom JSON type that accepts either a float (seconds, e.g. 1.5 = 1s500ms)
// or a string parseable by [time.ParseDuration] (e.g. "5s", "100ms").
type sessionDuration struct{ time.Duration }

func (d *sessionDuration) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		d.Duration = time.Duration(f * float64(time.Second))

		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("delay must be a number (seconds) or a duration string (e.g. \"5s\"): %w", err)
	}

	parsed, parseErr := time.ParseDuration(s)
	if parseErr != nil {
		return fmt.Errorf("invalid duration %q: %w", s, parseErr)
	}

	d.Duration = parsed

	return nil
}

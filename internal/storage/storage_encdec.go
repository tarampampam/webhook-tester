package storage

import (
	"encoding/json"
	"fmt"
	"time"
)

var (
	_ json.Marshaler   = (*Time)(nil) // ensure that Time implements the json.Marshaler interface
	_ json.Unmarshaler = (*Time)(nil) // ensure that Time implements the json.Unmarshaler interface
)

// MarshalJSON implements the json.Marshaler interface and returns the time in unix-nano format.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("0"), nil
	}

	return []byte(fmt.Sprintf("%d", t.Time.UnixNano())), nil // fmt.Sprintf used here to avoid exponential notation
}

// UnmarshalJSON implements the json.Unmarshaler interface and parses the time in unix-nano format.
func (t *Time) UnmarshalJSON(data []byte) error {
	var unixNano int64
	if err := json.Unmarshal(data, &unixNano); err != nil {
		return err
	}

	if unixNano == 0 {
		t.Time = time.Time{}

		return nil
	}

	t.Time = time.Unix(0, unixNano)

	return nil
}

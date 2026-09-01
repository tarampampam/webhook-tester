package j

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncodeTo encodes a value of type T as JSON and writes it to the provided writer.
//
// Note: [json.Encoder] adds a newline after encoding, so the output will always end with a newline character.
func EncodeTo[T comparable](w io.Writer, v T) error {
	if w == nil {
		return errors.New("nil writer")
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	return nil
}

// Encode encodes a value of type T as JSON and returns the resulting byte slice.
func Encode[T comparable](v T) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encoding json: %w", err)
	}

	return b, nil
}

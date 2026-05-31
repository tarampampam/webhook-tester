package j

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidJSON is returned when the JSON body is invalid, cannot be decoded into the expected struct, or contains
// unexpected fields or trailing data.
var ErrInvalidJSON = errors.New("invalid json")

// invalidJsonError is a custom error type that wraps an underlying error (e.g. from decoding or closing) and
// implements error unwrapping and comparison to ErrInvalidJSON. It allows us to return detailed error information
// (via Unwrap's cause) while still enabling callers to check for ErrInvalidJSON using [errors.Is].
type invalidJsonError struct{ cause error }

var (
	_ error                       = (*invalidJsonError)(nil) //nolint:errcheck
	_ interface{ Unwrap() error } = (*invalidJsonError)(nil) //nolint:errcheck
	_ interface{ Is(error) bool } = (*invalidJsonError)(nil) //nolint:errcheck
)

// Error implements the error interface.
func (err *invalidJsonError) Error() string {
	return fmt.Sprintf("%v: %v", ErrInvalidJSON, err.cause)
}

// Unwrap allows [errors.Unwrap] to retrieve the underlying error.
func (err *invalidJsonError) Unwrap() error { return err.cause }

// Is allows [errors.Is] to compare invalidJsonError with ErrInvalidJSON.
func (err *invalidJsonError) Is(target error) bool { return target == ErrInvalidJSON }

// Decode decodes a JSON body from the provided [io.ReadCloser] into a struct of type T.
//
// It ensures that the request body is CLOSED after reading (even if an error occurs) and that
// the JSON is strictly validated (no unknown fields, no trailing data).
//
// If the JSON is invalid, the returned error wraps ErrInvalidJSON and can be detected via [errors.Is]. Additionally,
// it may wrap other errors (e.g. from decoding or closing) which can be inspected with [errors.Unwrap] or [errors.Is].
func Decode[T any](r io.ReadCloser) (_ *T, outErr error) {
	if r == nil {
		return nil, errors.New("nil request body") // error NOT wrapped since this is a caller error
	}

	// ensure the request body is closed at the end of this function (if closing fails, append that
	// error to the output error (if any))
	defer func() {
		if err := r.Close(); err != nil {
			if outErr == nil {
				outErr = err
			} else {
				outErr = fmt.Errorf("%w: closing: %w", outErr, err)
			}
		}
	}()

	var payload T // payload will hold the decoded JSON structure

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields() // strict JSON: reject unexpected keys

	// decode the JSON body into the payload
	if err := dec.Decode(&payload); err != nil {
		// colon is omitted here to format the error message as "invalid json: decoding json: <cause>"
		// instead of "invalid json: decoding: json: <cause>"
		return nil, &invalidJsonError{cause: fmt.Errorf("decoding %w", err)} // invalid JSON or type mismatch
	}

	// guard against trailing data (multiple JSON objects in the body)
	var extra json.RawMessage

	// check if there is any trailing data after the first JSON object - this prevents
	// clients from sending multiple JSON objects in one body
	if err := dec.Decode(&extra); err != nil {
		// if we reached EOF immediately after the first JSON, it's valid
		if !errors.Is(err, io.EOF) {
			// any decoding error (other than EOF) after the first object is invalid
			return nil, &invalidJsonError{cause: fmt.Errorf("multiple JSON objects in body: %w", err)}
		}

		// successfully reached EOF - return the validated payload
		return &payload, nil
	}

	// if decoding succeeded (no error), then extra JSON exists after the main object - reject it
	return nil, &invalidJsonError{cause: errors.New("unexpected extra JSON in body")}
}

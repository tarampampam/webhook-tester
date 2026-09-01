package openapi

import (
	"errors"
)

// Following errors allows handlers to communicate a domain error that is consistently translated to an
// HTTP status code in the OpenAPI adapter.
var (
	ErrBadRequest  = errors.New("bad request")  // 400 Bad Request
	ErrNotFound    = errors.New("not found")    // 404 Not Found
	ErrServerError = errors.New("server error") // 500 Internal Server Error
)

// apiError is a typed error that wraps a sentinel error with a custom message.
type apiError struct {
	msg      string
	sentinel error
}

var (
	_ error                       = (*apiError)(nil) //nolint:errcheck
	_ interface{ Unwrap() error } = (*apiError)(nil) //nolint:errcheck
	_ interface{ Is(error) bool } = (*apiError)(nil) //nolint:errcheck
)

// Error implements the error interface.
func (e *apiError) Error() string { return e.msg }

// Is allows [errors.Is] to compare apiError with its sentinel error.
func (e *apiError) Is(target error) bool { return target == e.sentinel }

// Unwrap allows [errors.Unwrap] to retrieve the underlying sentinel error.
func (e *apiError) Unwrap() error { return e.sentinel }

// NewErrNotFound creates a new error with the given message that satisfies [errors.Is] for [ErrNotFound].
func NewErrNotFound(msg string) error { return &apiError{msg: msg, sentinel: ErrNotFound} }

// NewErrBadRequest creates a new error with the given message that satisfies [errors.Is] for [ErrBadRequest].
func NewErrBadRequest(msg string) error { return &apiError{msg: msg, sentinel: ErrBadRequest} }

// NewErrServerError creates a new error with the given message that satisfies [errors.Is] for [ErrServerError].
func NewErrServerError(msg string) error { return &apiError{msg: msg, sentinel: ErrServerError} }

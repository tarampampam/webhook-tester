package openapi_test

import (
	"errors"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestNewErrConstructors(t *testing.T) {
	t.Parallel()

	allSentinels := []error{
		openapi.ErrNotFound,
		openapi.ErrBadRequest,
		openapi.ErrServerError,
	}

	for name, tc := range map[string]struct {
		newErr   func(string) error
		sentinel error
	}{
		"NewErrNotFound":    {newErr: openapi.NewErrNotFound, sentinel: openapi.ErrNotFound},
		"NewErrBadRequest":  {newErr: openapi.NewErrBadRequest, sentinel: openapi.ErrBadRequest},
		"NewErrServerError": {newErr: openapi.NewErrServerError, sentinel: openapi.ErrServerError},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for msg, wantMsg := range map[string]string{
				"non-empty message": "some error",
				"empty message":     "",
			} {
				t.Run(msg, func(t *testing.T) {
					err := tc.newErr(wantMsg)

					assert.ErrorEqual(t, err, wantMsg)
					assert.True(t, errors.Is(err, tc.sentinel))

					for _, other := range allSentinels {
						if !errors.Is(other, tc.sentinel) {
							assert.False(t, errors.Is(err, other))
						}
					}
				})
			}
		})
	}
}

func TestSentinelErrors_NotWrappingEachOther(t *testing.T) { //nolint:tparallel
	t.Parallel()

	for name, tc := range map[string]struct {
		giveErr    error
		wantNotErr error
	}{
		"ErrNotFound is not ErrBadRequest":    {giveErr: openapi.ErrNotFound, wantNotErr: openapi.ErrBadRequest},
		"ErrNotFound is not ErrServerError":   {giveErr: openapi.ErrNotFound, wantNotErr: openapi.ErrServerError},
		"ErrBadRequest is not ErrNotFound":    {giveErr: openapi.ErrBadRequest, wantNotErr: openapi.ErrNotFound},
		"ErrBadRequest is not ErrServerError": {giveErr: openapi.ErrBadRequest, wantNotErr: openapi.ErrServerError},
		"ErrServerError is not ErrNotFound":   {giveErr: openapi.ErrServerError, wantNotErr: openapi.ErrNotFound},
		"ErrServerError is not ErrBadRequest": {giveErr: openapi.ErrServerError, wantNotErr: openapi.ErrBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			assert.False(t, errors.Is(tc.giveErr, tc.wantNotErr))
		})
	}
}

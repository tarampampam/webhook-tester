package version_latest_test

import (
	"context"
	"errors"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/version_latest"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		providerFn func(context.Context) (string, error)
		wantErr    bool
		checkErr   func(*testing.T, error)
		check      func(*testing.T, *openapi.VersionResponse)
	}{
		"success/version returned": {
			providerFn: func(_ context.Context) (string, error) { return "v1.2.3", nil },
			check: func(t *testing.T, resp *openapi.VersionResponse) {
				assert.Equal(t, "v1.2.3", resp.Version)
			},
		},

		"error/provider fails - error wrapped": {
			providerFn: func(_ context.Context) (string, error) {
				return "", errors.New("network failure")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to fetch the latest version")
				assert.ErrorContains(t, err, "network failure")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := version_latest.New(tt.providerFn)
			resp, err := h.Handle(t.Context())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)

				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				if tt.check != nil {
					tt.check(t, resp)
				}
			}
		})
	}

	t.Run("cache/hit - provider called once within TTL", func(t *testing.T) {
		t.Parallel()

		var calls int

		h := version_latest.New(func(_ context.Context) (string, error) {
			calls++
			return "v2.0.0", nil
		})

		resp1, err := h.Handle(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, "v2.0.0", resp1.Version)

		resp2, err := h.Handle(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, "v2.0.0", resp2.Version)

		assert.Equal(t, 1, calls)
	})

	t.Run("cache/empty version not cached", func(t *testing.T) {
		t.Parallel()

		var calls int

		h := version_latest.New(func(_ context.Context) (string, error) {
			calls++
			return "", nil
		})

		_, _ = h.Handle(t.Context())
		_, _ = h.Handle(t.Context())

		assert.Equal(t, 2, calls)
	})
}

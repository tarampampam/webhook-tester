package openapi_test

import (
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestSessionResponseOptions_Validate(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		give        openapi.SessionResponseOptions
		wantErr     bool
		errContains string
	}{
		"valid/minimal": {
			give: openapi.SessionResponseOptions{StatusCode: openapi.StatusCodeMin},
		},
		"valid/all fields at max": {
			give: openapi.SessionResponseOptions{
				StatusCode:         openapi.StatusCodeMax,
				Delay:              30,
				ResponseBodyBase64: strings.Repeat("a", openapi.Base64EncodedMaxLength),
				Headers:            []openapi.HttpHeader{{Name: "X", Value: strings.Repeat("v", 2048)}},
			},
		},
		"valid/multiple headers": {
			give: openapi.SessionResponseOptions{
				StatusCode: 200,
				Headers: []openapi.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: ""},
				},
			},
		},
		"valid/no headers": {
			give: openapi.SessionResponseOptions{StatusCode: 200, Headers: []openapi.HttpHeader{}},
		},

		"invalid/status_code below min": {
			give:        openapi.SessionResponseOptions{StatusCode: openapi.StatusCodeMin - 1},
			wantErr:     true,
			errContains: "status_code",
		},
		"invalid/status_code above max": {
			give:        openapi.SessionResponseOptions{StatusCode: openapi.StatusCodeMax + 1},
			wantErr:     true,
			errContains: "status_code",
		},
		"invalid/status_code zero": {
			give:        openapi.SessionResponseOptions{StatusCode: 0},
			wantErr:     true,
			errContains: "status_code",
		},
		"invalid/delay above max": {
			give:        openapi.SessionResponseOptions{StatusCode: 200, Delay: 31},
			wantErr:     true,
			errContains: "delay",
		},
		"invalid/body too long": {
			give: openapi.SessionResponseOptions{
				StatusCode:         200,
				ResponseBodyBase64: strings.Repeat("a", openapi.Base64EncodedMaxLength+1),
			},
			wantErr:     true,
			errContains: "response_body_base64",
		},
		"invalid/header name empty": {
			give: openapi.SessionResponseOptions{
				StatusCode: 200,
				Headers:    []openapi.HttpHeader{{Name: ""}},
			},
			wantErr:     true,
			errContains: "headers[0].name",
		},
		"invalid/header name too long": {
			give: openapi.SessionResponseOptions{
				StatusCode: 200,
				Headers:    []openapi.HttpHeader{{Name: strings.Repeat("a", 41)}},
			},
			wantErr:     true,
			errContains: "headers[0].name",
		},
		"invalid/header value too long": {
			give: openapi.SessionResponseOptions{
				StatusCode: 200,
				Headers:    []openapi.HttpHeader{{Name: "X", Value: strings.Repeat("v", 2049)}},
			},
			wantErr:     true,
			errContains: "headers[0].value",
		},
		"invalid/second header name empty": {
			give: openapi.SessionResponseOptions{
				StatusCode: 200,
				Headers: []openapi.HttpHeader{
					{Name: "Valid", Value: "ok"},
					{Name: ""},
				},
			},
			wantErr:     true,
			errContains: "headers[1].name",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tt.give.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, openapi.ErrValidationFailed)
				assert.ErrorContains(t, err, tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApiSessionRequestsSubscribeParams_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&openapi.ApiSessionRequestsSubscribeParams{}).Validate())
}

func TestIsValidUUID(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		give string
		want bool
	}{
		"valid v4":         {give: "550e8400-e29b-41d4-a716-446655440000", want: true},
		"valid v4 alt":     {give: "9b6bbab9-c197-4dd3-bc3f-3cb6253820c7", want: true},
		"empty":            {give: "", want: false},
		"too short":        {give: "550e8400-e29b-41d4-a716", want: false},
		"too long":         {give: "550e8400-e29b-41d4-a716-446655440000-extra", want: false},
		"wrong format":     {give: "not-a-uuid-at-all-here", want: false},
		"all zeros":        {give: "00000000-0000-0000-0000-000000000000", want: true},
		"wrong separators": {give: "550e8400/e29b/41d4/a716/446655440000", want: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, openapi.IsValidUUID(tt.give))
		})
	}
}

func TestValidateUUIDs(t *testing.T) {
	t.Parallel()

	const (
		validUUID1 = "550e8400-e29b-41d4-a716-446655440000"
		validUUID2 = "9b6bbab9-c197-4dd3-bc3f-3cb6253820c7"
		validUUID3 = "d74a7998-dcbc-4d77-82ba-27945e56a25d"
	)

	for name, tt := range map[string]struct {
		give        []string
		minLen      int
		maxLen      int
		wantErr     bool
		errContains string
	}{
		"valid/single": {
			give: []string{validUUID1}, minLen: 1, maxLen: 10,
		},
		"valid/multiple": {
			give: []string{validUUID1, validUUID2, validUUID3}, minLen: 1, maxLen: 10,
		},
		"valid/at min boundary": {
			give: []string{validUUID1}, minLen: 1, maxLen: 1,
		},
		"valid/at max boundary": {
			give: []string{validUUID1, validUUID2}, minLen: 1, maxLen: 2,
		},

		"invalid/too few": {
			give: []string{}, minLen: 1, maxLen: 10,
			wantErr: true, errContains: "number of ids",
		},
		"invalid/too many": {
			give: []string{validUUID1, validUUID2, validUUID3}, minLen: 1, maxLen: 2,
			wantErr: true, errContains: "number of ids",
		},
		"invalid/bad uuid at index 0": {
			give: []string{"not-a-uuid"}, minLen: 1, maxLen: 10,
			wantErr: true, errContains: "id[0]",
		},
		"invalid/bad uuid at index 1": {
			give: []string{validUUID1, "bad"}, minLen: 1, maxLen: 10,
			wantErr: true, errContains: "id[1]",
		},
		"invalid/empty uuid string": {
			give: []string{validUUID1, ""}, minLen: 1, maxLen: 10,
			wantErr: true, errContains: "id[1]",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := openapi.ValidateUUIDs(tt.give, tt.minLen, tt.maxLen)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, openapi.ErrValidationFailed)
				assert.ErrorContains(t, err, tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

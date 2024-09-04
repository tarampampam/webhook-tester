package api_test

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/internal/api"
)

func TestNewSession_Validate(t *testing.T) {
	assert.NoError(t, api.NewSession{}.Validate())

	var (
		lowCode         = 99
		highCode        = 531
		longContentType = strings.Repeat("x", 33)
		highDelay       = 31
		wrongBase64     = "foobar"
		longBase64      = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 10241)))
	)

	assert.True(t, len(longBase64) > 10240)

	for name, tt := range map[string]struct {
		give          api.NewSession
		wantErrSubstr string
	}{
		"too low status code": {
			give:          api.NewSession{StatusCode: &lowCode},
			wantErrSubstr: "wrong status code",
		},
		"too high status code": {
			give:          api.NewSession{StatusCode: &highCode},
			wantErrSubstr: "wrong status code",
		},
		"too long content-type": {
			give:          api.NewSession{ContentType: &longContentType},
			wantErrSubstr: "content-type value is too long",
		},
		"too high delay": {
			give:          api.NewSession{ResponseDelay: &highDelay},
			wantErrSubstr: "response delay is too much",
		},
		"wrong base64 body": {
			give:          api.NewSession{ResponseContentBase64: &wrongBase64},
			wantErrSubstr: "cannot decode response body",
		},
		"base64 body too long": {
			give:          api.NewSession{ResponseContentBase64: &longBase64},
			wantErrSubstr: "response content is too large",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := tt.give.Validate()

			if tt.wantErrSubstr != "" {
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewSession_Getters(t *testing.T) {
	// default values
	assert.EqualValues(t, http.StatusOK, api.NewSession{}.GetStatusCode())
	assert.EqualValues(t, "text/plain", api.NewSession{}.GetContentType())
	assert.EqualValues(t, 0, api.NewSession{}.GetResponseDelay())
	assert.EqualValues(t, []byte{}, api.NewSession{}.ResponseContent())

	var (
		code    = 123
		cType   = "foo"
		delay   = 10
		content = "Zm9v" // foo
	)

	var valid = api.NewSession{
		StatusCode:            &code,
		ContentType:           &cType,
		ResponseContentBase64: &content,
		ResponseDelay:         &delay,
	}

	assert.EqualValues(t, code, valid.GetStatusCode())
	assert.EqualValues(t, cType, valid.GetContentType())
	assert.EqualValues(t, delay, valid.GetResponseDelay())
	assert.EqualValues(t, "foo", valid.ResponseContent())
}

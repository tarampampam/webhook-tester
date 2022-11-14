package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/pkg/errors"
)

func (data NewSession) Validate() error {
	const (
		minStatusCode, maxStatusCode = StatusCode(100), StatusCode(530)
		maxContentTypeLength         = 32
		maxResponseContentLength     = 10240
		maxResponseDelaySeconds      = ResponseDelayInSeconds(30) // IMPORTANT! Must be less than http/writeTimeout value!
	)

	if data.StatusCode != nil && (*data.StatusCode < minStatusCode || *data.StatusCode > maxStatusCode) {
		return fmt.Errorf("wrong status code (should be between %d and %d)", minStatusCode, maxStatusCode)
	}

	if data.ContentType != nil && utf8.RuneCountInString(*data.ContentType) > maxContentTypeLength {
		return fmt.Errorf("content-type value is too long (max length is %d)", maxContentTypeLength)
	}

	if data.ResponseDelay != nil && *data.ResponseDelay > maxResponseDelaySeconds {
		return fmt.Errorf("response delay is too much (max is %d)", maxResponseDelaySeconds)
	}

	if data.ResponseContentBase64 != nil {
		if v, err := base64.StdEncoding.DecodeString(*data.ResponseContentBase64); err != nil {
			return errors.Wrap(err, "cannot decode response body (wrong base64)")
		} else if utf8.RuneCount(v) > maxResponseContentLength {
			return fmt.Errorf("response content is too large (max length is %d)", maxResponseContentLength)
		}
	}

	return nil
}

func (data NewSession) GetStatusCode() uint16 {
	if data.StatusCode != nil {
		return uint16(*data.StatusCode)
	}

	return http.StatusOK // default value
}

func (data NewSession) GetContentType() string {
	if data.ContentType != nil {
		return *data.ContentType
	}

	return "text/plain" // default value
}

func (data NewSession) GetResponseDelay() int {
	if data.ResponseDelay != nil {
		return *data.ResponseDelay
	}

	return 0 // default value
}

func (data NewSession) ResponseContent() []byte {
	if data.ResponseContentBase64 != nil {
		if v, err := base64.StdEncoding.DecodeString(*data.ResponseContentBase64); err == nil {
			return v
		}
	}

	return []byte{} // default value
}

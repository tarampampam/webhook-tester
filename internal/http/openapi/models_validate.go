package openapi

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

func (data CreateSessionRequest) Validate() error {
	const (
		maxDelaySeconds                  = 30 // IMPORTANT! Must be less than http/writeTimeout value!
		maxHeadersCount                  = 10
		minHeaderKeyLen, maxHeaderKeyLen = 1, 40
		maxHeaderValueLen                = 2048
		maxResponseBodyLen               = 10240
		minStatusCode, maxStatusCode     = StatusCode(200), StatusCode(530)
	)

	if data.Delay > maxDelaySeconds {
		return fmt.Errorf("response delay is too much (max is %d)", maxDelaySeconds)
	}

	if len(data.Headers) > maxHeadersCount {
		return fmt.Errorf("too many headers (max count is %d)", maxHeadersCount)
	}

	for _, header := range data.Headers {
		if l := utf8.RuneCountInString(header.Name); l < minHeaderKeyLen || l > maxHeaderKeyLen {
			return fmt.Errorf("header key length should be between %d and %d", minHeaderKeyLen, maxHeaderKeyLen)
		}

		if strings.TrimSpace(header.Name) == "" {
			return fmt.Errorf("header key should not be empty")
		}

		if l := utf8.RuneCountInString(header.Value); l > maxHeaderValueLen {
			return fmt.Errorf("header value length should be less than %d", maxHeaderValueLen)
		}
	}

	if v, err := base64.StdEncoding.DecodeString(data.ResponseBodyBase64); err != nil {
		return fmt.Errorf("cannot decode response body (wrong base64): %w", err)
	} else if utf8.RuneCount(v) > maxResponseBodyLen {
		return fmt.Errorf("response content is too large (max length is %d)", maxResponseBodyLen)
	}

	if data.StatusCode < minStatusCode || data.StatusCode > maxStatusCode {
		return fmt.Errorf("wrong status code (should be between %d and %d)", minStatusCode, maxStatusCode)
	}

	return nil
}

package openapi

import (
	"fmt"

	"github.com/google/uuid"
)

// Validate checks if the SessionResponseOptions fields are within the ranges defined in the OpenAPI spec.
func (o *SessionResponseOptions) Validate() (outErr error) {
	defer func() { outErr = wrapValidationError(outErr) }() // to guarantee that the error returned is always wrapped

	const (
		minCode                  = StatusCodeMin
		maxCode                  = StatusCodeMax
		maxDelay          uint16 = 30 // from the OpenAPI spec (SessionResponseOptions.delay.maximum = 30)
		maxBodyLen               = Base64EncodedMaxLength
		minHeaderNameLen         = 1    // from the OpenAPI spec (HttpHeader.name.minLength = 1)
		maxHeaderNameLen         = 40   // from the OpenAPI spec (HttpHeader.name.maxLength = 40)
		maxHeaderValueLen        = 2048 // from the OpenAPI spec (HttpHeader.value.maxLength = 2048)
	)

	if o.StatusCode < minCode || o.StatusCode > maxCode {
		return fmt.Errorf("wrong status code: must be in [%d, %d]", minCode, maxCode)
	}

	if o.Delay > maxDelay {
		return fmt.Errorf("delay must be at most %d", maxDelay)
	}

	if len(o.ResponseBodyBase64) > maxBodyLen {
		return fmt.Errorf("response_body_base64 length must be at most %d", maxBodyLen)
	}

	for i, h := range o.Headers {
		if len(h.Name) < minHeaderNameLen || len(h.Name) > maxHeaderNameLen {
			return fmt.Errorf("header key length must be in [%d, %d] (headers[%d].name)", minHeaderNameLen, maxHeaderNameLen, i)
		}

		if len(h.Value) > maxHeaderValueLen {
			return fmt.Errorf("header value length must be at most %d (headers[%d].value)", maxHeaderValueLen, i)
		}
	}

	return nil
}

// Validate delegates WebSocket handshake header validation to the WebSocket upgrader library.
func (p *ApiSessionRequestsSubscribeParams) Validate() error { return nil }

// IsValidUUID checks if passed string is valid UUID v4.
func IsValidUUID(id string) bool {
	if len(id) != 36 { //nolint:mnd // the length of a UUID string (e.g. "550e8400-e29b-41d4-a716-446655440000")
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}

// ValidateUUIDs checks if the number of IDs is within the specified range and if each ID is a valid UUID v4.
func ValidateUUIDs[T ~string](ids []T, minLen, maxLen int) (outErr error) {
	defer func() { outErr = wrapValidationError(outErr) }() // to guarantee that the error returned is always wrapped

	if len(ids) < minLen || len(ids) > maxLen {
		return fmt.Errorf("number of ids must be in [%d, %d]", minLen, maxLen)
	}

	for i, id := range ids {
		if !IsValidUUID(string(id)) {
			return fmt.Errorf("id[%d] is not a valid UUID", i)
		}
	}

	return nil
}

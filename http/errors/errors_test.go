package errors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerError(t *testing.T) {
	err := NewServerError(1, "foo")

	assert.False(t, err.Success)
	assert.Equal(t, uint16(1), err.Code)
	assert.Equal(t, "foo", err.Message)
}

func TestServerError_JsonCasting(t *testing.T) {
	t.Parallel()

	err := ServerError{
		Code:    123,
		Success: true,
		Message: "foo",
	}

	asJSON, _ := json.Marshal(err)

	assert.JSONEq(t, `{"code":123,"success":true,"message":"foo"}`, string(asJSON))
	assert.JSONEq(t, `{"code":123,"success":true,"message":"foo"}`, string(err.ToJSON()))
}

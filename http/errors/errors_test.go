package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJSONError(t *testing.T) {
	err := NewJSONError(1, "foo")

	assert.True(t, err.Error)
	assert.Equal(t, uint16(1), err.Code)
	assert.Equal(t, "foo", err.Message)
}

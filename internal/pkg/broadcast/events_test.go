package broadcast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestRegisteredEvent(t *testing.T) {
	e := NewRequestRegisteredEvent("foo")

	assert.Equal(t, "foo", e.Data())
	assert.Equal(t, "request-registered", e.Name())
}

func TestNewRequestDeletedEvent(t *testing.T) {
	e := NewRequestDeletedEvent("foo")

	assert.Equal(t, "foo", e.Data())
	assert.Equal(t, "request-deleted", e.Name())
}

func TestNewAllRequestsDeletedEvent(t *testing.T) {
	e := NewAllRequestsDeletedEvent()

	assert.Equal(t, "*", e.Data())
	assert.Equal(t, "requests-deleted", e.Name())
}

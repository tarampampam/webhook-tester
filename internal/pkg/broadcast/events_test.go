package broadcast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
)

func TestNewRequestRegisteredEvent(t *testing.T) {
	e := broadcast.NewRequestRegisteredEvent("foo")

	assert.Equal(t, "foo", e.Data())
	assert.Equal(t, "request-registered", e.Name())
}

func TestNewRequestDeletedEvent(t *testing.T) {
	e := broadcast.NewRequestDeletedEvent("foo")

	assert.Equal(t, "foo", e.Data())
	assert.Equal(t, "request-deleted", e.Name())
}

func TestNewAllRequestsDeletedEvent(t *testing.T) {
	e := broadcast.NewAllRequestsDeletedEvent()

	assert.Equal(t, "*", e.Data())
	assert.Equal(t, "requests-deleted", e.Name())
}

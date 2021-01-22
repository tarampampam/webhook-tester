package storage

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUUID(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := NewUUID()
		_, err := uuid.Parse(s)
		assert.Nil(t, err)
	}
}

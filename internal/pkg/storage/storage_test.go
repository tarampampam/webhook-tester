package storage_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestNewUUID(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := storage.NewUUID()
		_, err := uuid.Parse(s)
		assert.Nil(t, err)
	}
}

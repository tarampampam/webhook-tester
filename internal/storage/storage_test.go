package storage_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/internal/storage"
)

func TestNewUUID(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := storage.NewUUID()
		_, err := uuid.Parse(s)
		assert.Nil(t, err)
	}
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, storage.IsValidUUID("00000000-0000-0000-0000-000000000000"))
	assert.True(t, storage.IsValidUUID("9b6bbab9-c197-4dd3-bc3f-3cb6253820c7"))

	assert.False(t, storage.IsValidUUID("9b6bbab9-c197-4dd3-bc3f-3cb6253820ZZ"))
	assert.False(t, storage.IsValidUUID("ZZ6bbab9-c197-4dd3-bc3f-3cb6253820c7"))
	assert.False(t, storage.IsValidUUID("00-00-00-00-00"))
}

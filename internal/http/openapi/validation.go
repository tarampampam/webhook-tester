package openapi

import "github.com/google/uuid"

const UUIDLength = 36

// IsValidUUID checks if passed string is valid UUID v4.
func IsValidUUID(id string) bool {
	if len(id) != UUIDLength {
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}

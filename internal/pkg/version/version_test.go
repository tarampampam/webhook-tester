package version_test

import (
	"testing"

	"github.com/tarampampam/webhook-tester/internal/pkg/version"
)

func TestVersion(t *testing.T) {
	if value := version.Version(); value != "0.0.0@undefined" {
		t.Errorf("Unexpected default version value: %s", value)
	}
}

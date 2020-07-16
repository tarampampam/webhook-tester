package version

import "testing"

func TestVersion(t *testing.T) {
	t.Parallel()

	if value := Version(); value != "undefined@undefined" {
		t.Errorf("Unexpected default version value: %s", value)
	}
}

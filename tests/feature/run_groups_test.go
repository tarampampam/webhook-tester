package feature_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"

	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

// TestWithMemory runs the full suite with the default in-memory storage and pub/sub drivers.
func TestWithMemory(t *testing.T) {
	t.Parallel()

	runGroupTests(t, ft.MustRunApp(t))
}

// TestWithRedis runs the full suite with Redis-backed storage and pub/sub drivers via miniredis.
func TestWithRedis(t *testing.T) {
	t.Parallel()

	runGroupTests(t, ft.MustRunApp(t,
		"--storage-driver", "redis",
		"--pubsub-driver", "redis",
		"--redis-dsn", "redis://"+miniredis.RunT(t).Addr(),
	))
}

// TestWithFS runs the full suite with the filesystem storage driver.
func TestWithFS(t *testing.T) {
	t.Parallel()

	runGroupTests(t, ft.MustRunApp(t,
		"--storage-driver", "fs",
		"--fs-storage-dir", t.TempDir(),
	))
}

// runGroupTests runs the full feature test suite against a running application at baseUrl.
func runGroupTests(t *testing.T, baseUrl string) {
	t.Helper()

	t.Run("spa", func(t *testing.T) {
		t.Parallel()
		groupTestSPA(t, baseUrl)
	})

	t.Run("api", func(t *testing.T) {
		t.Parallel()
		groupTestAPI(t, baseUrl)
	})

	t.Run("session", func(t *testing.T) {
		t.Parallel()
		groupTestAPISession(t, baseUrl)
	})

	t.Run("requests", func(t *testing.T) {
		t.Parallel()
		groupTestWebhook(t, baseUrl)
	})
}

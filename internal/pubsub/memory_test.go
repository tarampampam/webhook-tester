package pubsub_test

import (
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
)

func memoryFactory(tb testing.TB) pubsub.PubSub {
	tb.Helper()

	ps := pubsub.NewMemory()

	tb.Cleanup(func() { _ = ps.Close() })

	return ps
}

func TestMemory(t *testing.T) { t.Parallel(); RunSuite(t, memoryFactory) }

func BenchmarkMemory(b *testing.B) { RunBenchmarks(b, memoryFactory) }

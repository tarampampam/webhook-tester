package storage_test

import (
	"context"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func memoryFactory(tb testing.TB, limit uint, timeNow func() time.Time) storage.Storage {
	tb.Helper()

	m := storage.NewMemory(tb.Context(), limit, storage.WithMemoryTimeNow(timeNow))
	tb.Cleanup(func() { _ = m.Close() })

	return m
}

func TestMemory(t *testing.T) { t.Parallel(); RunSuite(t, memoryFactory) } //nolint:gci

func TestMemory_ContextCancellationClosesStorage(t *testing.T) {
	t.Parallel()
	testContextCancellationClosesStorage(t, func(ctx context.Context) storage.Storage {
		m := storage.NewMemory(ctx, 10)
		t.Cleanup(func() { _ = m.Close() })

		return m
	})
}

func TestMemory_CleanupEvictsExpired(t *testing.T) {
	t.Parallel()

	ft := newFakeTime()

	m := storage.NewMemory(
		t.Context(),
		10,
		storage.WithMemoryTimeNow(ft.Now),
		storage.WithMemoryCleanupInterval(time.Microsecond),
	)
	defer func() { _ = m.Close() }()

	_, err := m.NewSession(t.Context(), "s1", storage.SessionResponse{}, time.Second)
	assert.NoError(t, err)

	_, err = m.GetSession(t.Context(), "s1")
	assert.NoError(t, err)

	ft.Advance(2 * time.Second) // expire the session

	time.Sleep(20 * time.Millisecond) // let the cleanup goroutine fire

	_, err = m.GetSession(t.Context(), "s1")
	assert.ErrorIs(t, err, storage.ErrSessionNotFound)
}

func BenchmarkMemory(b *testing.B) { RunBenchmarks(b, memoryFactory) }

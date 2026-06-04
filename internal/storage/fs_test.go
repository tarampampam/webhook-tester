package storage_test

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"os"
	"path"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func fsFactory(tb testing.TB, limit uint, timeNow func() time.Time) storage.Storage {
	tb.Helper()

	root, err := os.OpenRoot(tb.TempDir())
	if err != nil {
		tb.Fatalf("open root: %v", err)
	}

	s := storage.NewFS(
		tb.Context(),
		root,
		limit,
		storage.WithFSTimeNow(timeNow),
		storage.WithFSCleanupInterval(10*time.Millisecond),
	)

	tb.Cleanup(func() {
		_ = s.Close()
		_ = root.Close()
	})

	return s
}

func TestFS(t *testing.T) { t.Parallel(); RunSuite(t, fsFactory) } //nolint:gci

func TestFS_ContextCancellationClosesStorage(t *testing.T) {
	t.Parallel()
	testContextCancellationClosesStorage(t, func(ctx context.Context) storage.Storage {
		root, err := os.OpenRoot(t.TempDir())
		assert.NoError(t, err)
		t.Cleanup(func() { _ = root.Close() })

		s := storage.NewFS(ctx, root, 10)

		t.Cleanup(func() { _ = s.Close() })

		return s
	})
}

func TestFS_CleanupEvictsExpired(t *testing.T) {
	t.Parallel()

	ft := newFakeTime()

	root, err := os.OpenRoot(t.TempDir())
	assert.NoError(t, err)

	defer func() { _ = root.Close() }()

	s := storage.NewFS(
		t.Context(),
		root,
		10,
		storage.WithFSTimeNow(ft.Now),
		storage.WithFSCleanupInterval(time.Microsecond),
	)
	defer func() { _ = s.Close() }()

	_, err = s.NewSession(t.Context(), "s1", storage.SessionResponse{}, time.Second)
	assert.NoError(t, err)

	_, err = s.GetSession(t.Context(), "s1")
	assert.NoError(t, err)

	ft.Advance(2 * time.Second) // expire the session

	time.Sleep(50 * time.Millisecond) // let the cleanup goroutine fire

	_, err = s.GetSession(t.Context(), "s1")
	assert.ErrorIs(t, err, storage.ErrSessionNotFound)
}

func TestFS_PersistenceSuite(t *testing.T) {
	t.Parallel()

	root, err := os.OpenRoot(t.TempDir())
	assert.NoError(t, err)

	t.Cleanup(func() { _ = root.Close() })

	RunPersistenceSuite(t, func(tb testing.TB, limit uint, timeNow func() time.Time) storage.Storage {
		s := storage.NewFS(
			tb.Context(),
			root,
			limit,
			storage.WithFSTimeNow(timeNow),
			storage.WithFSCleanupInterval(10*time.Millisecond),
		)
		tb.Cleanup(func() { _ = s.Close() })

		return s
	})
}

func TestFS_ReindexRecoversRequests(t *testing.T) {
	t.Parallel()

	root, err := os.OpenRoot(t.TempDir())
	assert.NoError(t, err)

	defer func() { _ = root.Close() }()

	s := storage.NewFS(t.Context(), root, 10)
	defer func() { _ = s.Close() }()

	_, err = s.NewSession(t.Context(), "s1", storage.SessionResponse{Code: 200}, storage.NoExpiration)
	assert.NoError(t, err)

	_, err = s.NewRequest(t.Context(), "s1", "r1", storage.CapturedRequest{Method: "GET"})
	assert.NoError(t, err)

	// simulate crash: remove the request index file
	h := md5.Sum([]byte("s1")) //nolint:gosec
	assert.NoError(t, root.Remove(path.Join(hex.EncodeToString(h[:]), "requests", "index.txt")))

	// session is still accessible - GetSession reads meta.bin directly, no index needed
	_, err = s.GetSession(t.Context(), "s1")
	assert.NoError(t, err)

	// request is not visible without the index
	reqSq, reqSqErr := s.GetRequests(t.Context(), "s1", nil)
	assert.NoError(t, reqSqErr)
	assert.Seq2Count(t, 0, reqSq)

	// Reindex restores the request index
	assert.NoError(t, s.Reindex(t.Context()))

	reqSq, reqSqErr = s.GetRequests(t.Context(), "s1", nil)
	assert.NoError(t, reqSqErr)
	assert.Seq2Count(t, 1, reqSq)
}

func TestFS_ReindexCancelledContextReturnsError(t *testing.T) {
	t.Parallel()

	root, err := os.OpenRoot(t.TempDir())
	assert.NoError(t, err)

	defer func() { _ = root.Close() }()

	s := storage.NewFS(t.Context(), root, 10)
	defer func() { _ = s.Close() }()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	assert.ErrorIs(t, s.Reindex(ctx), context.Canceled)
}

func BenchmarkFS(b *testing.B) { RunBenchmarks(b, fsFactory) }

package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func TestInMemory_Session_CreateReadDelete(t *testing.T) {
	t.Parallel()

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage { return storage.NewInMemory(sTTL, maxReq) },
		func(t time.Duration) { <-time.After(t) },
	)
}

func TestInMemory_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	testRequestCreateReadDelete(t, func(sTTL time.Duration, maxReq uint32) storage.Storage {
		return storage.NewInMemory(sTTL, maxReq)
	})
}

func TestInMemory_Close(t *testing.T) {
	t.Parallel()

	var ctx = context.Background()

	impl := storage.NewInMemory(time.Minute, 1)
	require.NoError(t, impl.Close())
	require.ErrorIs(t, impl.Close(), storage.ErrClosed) // second close

	_, err := impl.NewSession(ctx, storage.Session{})
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetSession(ctx, "foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteSession(ctx, "foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.NewRequest(ctx, "foo", storage.Request{})
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetRequest(ctx, "foo", "bar")
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetAllRequests(ctx, "foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteRequest(ctx, "foo", "bar")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteAllRequests(ctx, "foo")
	require.ErrorIs(t, err, storage.ErrClosed)
}

func TestInMemory_RaceProvocation(t *testing.T) {
	t.Parallel()

	testRaceProvocation(t, func(sTTL time.Duration, maxReq uint32) storage.Storage {
		return storage.NewInMemory(sTTL, maxReq, storage.WithInMemoryCleanupInterval(10*time.Nanosecond))
	})
}

// cpu: 12th Gen Intel(R) Core(TM) i7-1260P
// BenchmarkInMemory
// BenchmarkInMemory-16    	  400557	      3742 ns/op
func BenchmarkInMemory(b *testing.B) {
	s := storage.NewInMemory(time.Second, 10)
	defer s.Close()

	var ctx = context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sID, _ := s.NewSession(ctx, storage.Session{})
		_, _ = s.GetSession(ctx, sID)

		rID, _ := s.NewRequest(ctx, sID, storage.Request{})
		_, _ = s.GetRequest(ctx, sID, rID)

		_ = s.DeleteRequest(ctx, sID, rID)
	}
}

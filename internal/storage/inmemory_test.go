package storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/internal/storage"
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

	impl := storage.NewInMemory(time.Minute, 1)
	require.NoError(t, impl.Close())
	require.ErrorIs(t, impl.Close(), storage.ErrClosed) // second close

	_, err := impl.NewSession(storage.Session{})
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetSession("foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteSession("foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.NewRequest("foo", storage.Request{})
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetRequest("foo", "bar")
	require.ErrorIs(t, err, storage.ErrClosed)

	_, err = impl.GetAllRequests("foo")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteRequest("foo", "bar")
	require.ErrorIs(t, err, storage.ErrClosed)

	err = impl.DeleteAllRequests("foo")
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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sID, _ := s.NewSession(storage.Session{})
		_, _ = s.GetSession(sID)

		rID, _ := s.NewRequest(sID, storage.Request{})
		_, _ = s.GetRequest(sID, rID)

		_ = s.DeleteRequest(sID, rID)
	}
}

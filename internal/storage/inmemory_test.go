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

	var ft = newFakeTime()

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewInMemory(sTTL, maxReq, storage.WithInMemoryTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t) },
	)
}

func TestInMemory_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var ft = newFakeTime()

	testRequestCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewInMemory(sTTL, maxReq, storage.WithInMemoryTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t) },
	)
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

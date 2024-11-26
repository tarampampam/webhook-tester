package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func TestFS_Session_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var ft = newFakeTime()

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewFS(t.TempDir(), sTTL, maxReq, storage.WithFSTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t); <-time.After(t) },
	)
}

func TestFS_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var ft = newFakeTime()

	testRequestCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewFS(t.TempDir(), sTTL, maxReq, storage.WithFSTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t); <-time.After(t) },
	)
}

func TestFS_Close(t *testing.T) {
	t.Parallel()

	var ctx = context.Background()

	impl := storage.NewFS(t.TempDir(), time.Minute, 1)
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

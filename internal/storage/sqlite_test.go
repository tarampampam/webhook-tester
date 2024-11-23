package storage_test

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func TestSQLite_Session_CreateReadDelete(t *testing.T) {
	t.Parallel()

	// create a new SQLite database
	var sqlite = newSqliteDb(t)

	// migrate the database
	require.NoError(t, storage.NewSQLite(sqlite, 0, 0).Migrate(context.Background()))

	var ft = newFakeTime()

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewSQLite(sqlite, sTTL, maxReq, storage.WithSQLiteTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t) },
	)
}

func TestSQLite_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	// create a new SQLite database
	var sqlite = newSqliteDb(t)

	// migrate the database
	require.NoError(t, storage.NewSQLite(sqlite, 0, 0).Migrate(context.Background()))

	var ft = newFakeTime()

	testRequestCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewSQLite(sqlite, sTTL, maxReq, storage.WithSQLiteTimeNow(ft.Get))
		},
		func(t time.Duration) { ft.Add(t) },
	)
}

func TestSQLite_Close(t *testing.T) {
	t.Parallel()

	var ctx = context.Background()

	impl := storage.NewSQLite(newSqliteDb(t), time.Minute, 1)
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

func newSqliteDb(t *testing.T) *sql.DB {
	var sbFile = path.Join(t.TempDir(), "sqlite.db")

	db, dbErr := sql.Open("sqlite3", fmt.Sprintf("file:%s", sbFile)) // &_journal_mode=WAL&_txlock=immediate
	require.NoError(t, dbErr)

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}

//	func TestSQLite_RaceProvocation(t *testing.T) {
//		t.Parallel()
//
//		// create a new SQLite database
//		var sqlite = newSqliteDb(t)
//
//		// migrate the database
//		require.NoError(t, storage.NewSQLite(sqlite, 0, 0).Migrate(context.Background()))
//
//		testRaceProvocation(t, func(sTTL time.Duration, maxReq uint32) storage.Storage {
//			s := storage.NewSQLite(sqlite, sTTL, maxReq, storage.WithSQLiteCleanupInterval(10*time.Nanosecond))
//
//			require.NoError(t, s.Migrate(context.Background()))
//
//			return s
//		})
//	}

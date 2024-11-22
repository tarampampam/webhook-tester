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

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			var sbFile = path.Join(t.TempDir(), "sqlite.db")

			db, dbErr := sql.Open("sqlite3", fmt.Sprintf("file:%s", sbFile))
			require.NoError(t, dbErr)

			t.Cleanup(func() { require.NoError(t, db.Close()) })

			s, err := storage.NewSQLite(db, sTTL, maxReq)
			require.NoError(t, err)

			require.NoError(t, s.Migrate(context.Background()))

			return s
		},
		func(t time.Duration) { <-time.After(t) },
	)
}

func TestSQLite_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	testRequestCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			var sbFile = path.Join(t.TempDir(), "sqlite.db")

			db, dbErr := sql.Open("sqlite3", fmt.Sprintf("file:%s", sbFile))
			require.NoError(t, dbErr)

			t.Cleanup(func() { require.NoError(t, db.Close()) })

			s, err := storage.NewSQLite(db, sTTL, maxReq)
			require.NoError(t, err)

			require.NoError(t, s.Migrate(context.Background()))

			return s
		},
		func(t time.Duration) { <-time.After(t) },
	)
}

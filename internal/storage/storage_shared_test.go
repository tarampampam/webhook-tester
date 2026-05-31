package storage_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// StorageFactory creates a fresh, isolated storage instance for a single test or benchmark case.
// timeNow controls the instance's clock; limit caps requests stored per session (0 = unlimited).
// The factory must register tb.Cleanup to close the storage when the test or benchmark ends.
type StorageFactory func(tb testing.TB, limit uint, timeNow func() time.Time) storage.Storage

// RunSuite exercises the full [SessionStorage] + [RequestStorage] contract.
// Call it from each implementation's test file with an appropriate factory.
func RunSuite(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("NewSession", func(t *testing.T) { t.Parallel(); testNewSession(t, factory) })
	t.Run("GetSession", func(t *testing.T) { t.Parallel(); testGetSession(t, factory) })
	t.Run("AddSessionTTL", func(t *testing.T) { t.Parallel(); testAddSessionTTL(t, factory) })
	t.Run("DeleteSession", func(t *testing.T) { t.Parallel(); testDeleteSession(t, factory) })
	t.Run("NewRequest", func(t *testing.T) { t.Parallel(); testNewRequest(t, factory) })
	t.Run("GetRequest", func(t *testing.T) { t.Parallel(); testGetRequest(t, factory) })
	t.Run("GetRequests", func(t *testing.T) { t.Parallel(); testGetRequests(t, factory) })
	t.Run("DeleteRequest", func(t *testing.T) { t.Parallel(); testDeleteRequest(t, factory) })
	t.Run("DeleteAllRequests", func(t *testing.T) { t.Parallel(); testDeleteAllRequests(t, factory) })
	t.Run("SessionExpiry", func(t *testing.T) { t.Parallel(); testSessionExpiry(t, factory) })
	t.Run("RaceProvocation", func(t *testing.T) { t.Parallel(); testRaceProvocation(t, factory) })

	t.Run("Close", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		closer, ok := store.(io.Closer)
		if !ok {
			t.Skipf("storage %T does not implement io.Closer", store)

			return
		}

		assert.NoError(t, closer.Close())

		testClose(t, factory)
	})
}

const someSessionID = "some\\session Id"

func testNewSession(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("returned meta reflects creation time and TTL", func(t *testing.T) {
		t.Parallel()

		var (
			ft  = newFakeTime()
			now = ft.Now()
		)

		const ttl = time.Minute

		meta, err := factory(t, 10, ft.Now).NewSession(t.Context(), someSessionID, storage.SessionResponse{Code: 200}, ttl)
		assert.NoError(t, err)
		assert.NotNil(t, meta)

		assert.True(t, meta.CreatedAt.Equal(now))
		assert.True(t, meta.ExpiresAt.Equal(meta.CreatedAt.Add(ttl)))
	})

	t.Run("NoExpiration yields zero ExpiresAt", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		meta, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)
		assert.True(t, meta.ExpiresAt.IsZero())
	})

	t.Run("duplicate live session ID returns ErrSessionAlreadyExists", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.ErrorIs(t, err, storage.ErrSessionAlreadyExists)
		assert.ErrorIs(t, err, storage.ErrAlreadyExists) // ErrSessionAlreadyExists wraps ErrAlreadyExists
	})

	t.Run("expired session ID can be reused", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		const ttl = time.Minute

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, ttl)
		assert.NoError(t, err)

		ft.Advance(ttl + time.Millisecond)

		// creating a session with the same ID after expiry must succeed
		_, err = store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, ttl)
		assert.NoError(t, err)
	})
}

func testGetSession(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("returns correct Response and Meta", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
			resp  = storage.SessionResponse{
				Code:    201,
				Headers: []storage.ResponseHeader{{Name: "X-Foo", Value: "bar"}, {Name: "X-Baz", Value: "qux"}},
				Body:    []byte("hello body"),
				Delay:   3 * time.Second,
			}
			now = ft.Now()
		)

		const ttl = time.Minute

		_, err := store.NewSession(t.Context(), someSessionID, resp, ttl)
		assert.NoError(t, err)

		got, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.DeepEqual(t, resp, got.Response)
		assert.True(t, got.Meta.CreatedAt.Equal(now))
		assert.True(t, got.Meta.ExpiresAt.Equal(got.Meta.CreatedAt.Add(ttl)))
	})

	t.Run("unknown ID returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		got, err := factory(t, 10, time.Now).GetSession(t.Context(), "nonexistent")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returned session is a deep copy", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{Body: []byte("original")}, storage.NoExpiration)
		assert.NoError(t, err)

		got, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)

		got.Response.Body[0] = 'X' // mutate the returned copy

		// a subsequent read must still return the original data
		got2, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)
		assert.DeepEqual(t, []byte("original"), got2.Response.Body)
	})
}

func testAddSessionTTL(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("positive duration imposes expiration on an eternal session", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		const ttl = time.Minute

		now := ft.Now()

		assert.NoError(t, store.AddSessionTTL(t.Context(), someSessionID, ttl))

		// ExpiresAt must equal exactly now+ttl
		got, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)
		assert.True(t, got.Meta.ExpiresAt.Equal(now.Add(ttl)))

		ft.Advance(ttl + time.Millisecond)

		_, err = store.GetSession(t.Context(), someSessionID)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound) // past the new deadline
	})

	t.Run("NoExpiration removes the expiration deadline", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		const ttl = time.Minute

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, ttl)
		assert.NoError(t, err)

		// confirm the session has a deadline after creation
		before, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)
		assert.True(t, before.Meta.ExpiresAt.Equal(ft.Now().Add(ttl)))

		// clear the deadline
		assert.NoError(t, store.AddSessionTTL(t.Context(), someSessionID, storage.NoExpiration))

		// ExpiresAt must be zero — no deadline
		after, err := store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err)
		assert.True(t, after.Meta.ExpiresAt.IsZero())

		ft.Advance(time.Hour) // far past the original TTL

		_, err = store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err) // session is still present
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(
			t,
			factory(t, 10, time.Now).AddSessionTTL(t.Context(), "nonexistent", time.Hour),
			storage.ErrSessionNotFound,
		)
	})
}

func testDeleteSession(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("deleted session is no longer accessible", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		assert.NoError(t, store.DeleteSession(t.Context(), someSessionID))

		// subsequent reads must return ErrSessionNotFound
		_, err = store.GetSession(t.Context(), someSessionID)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(
			t,
			factory(t, 10, time.Now).DeleteSession(t.Context(), "nonexistent"),
			storage.ErrSessionNotFound,
		)
	})

	t.Run("its requests become inaccessible after deletion", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		assert.NoError(t, store.DeleteSession(t.Context(), someSessionID))

		_, err = store.GetRequests(t.Context(), someSessionID, nil)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
	})
}

func testNewRequest(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("returned meta reflects creation time", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
			now   = ft.Now()
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		meta, err := store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{Method: "POST"})
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.True(t, meta.CreatedAt.Equal(now))
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		_, err := factory(t, 10, time.Now).NewRequest(t.Context(), "nonexistent", "r1", storage.CapturedRequest{})
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("duplicate request ID returns ErrRequestAlreadyExists", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.ErrorIs(t, err, storage.ErrRequestAlreadyExists)
		assert.ErrorIs(t, err, storage.ErrAlreadyExists)
	})

	t.Run("oldest request is evicted when the per-session limit is reached", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 2, ft.Now) // IMPORTANT: limit = 2
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		ft.Advance(time.Second) // r2 must be strictly newer than r1 for the eviction order to be deterministic

		_, err = store.NewRequest(t.Context(), someSessionID, "r2", storage.CapturedRequest{})
		assert.NoError(t, err)

		ft.Advance(time.Second) // r3 must be strictly newer than r2

		// adding r3 reaches the cap; r1 (oldest) must be evicted, r2 and r3 must remain
		_, err = store.NewRequest(t.Context(), someSessionID, "r3", storage.CapturedRequest{})
		assert.NoError(t, err)

		_, err = store.GetRequest(t.Context(), someSessionID, "r1")
		assert.ErrorIs(t, err, storage.ErrRequestNotFound)

		_, err = store.GetRequest(t.Context(), someSessionID, "r2")
		assert.NoError(t, err)

		_, err = store.GetRequest(t.Context(), someSessionID, "r3")
		assert.NoError(t, err)
	})

	t.Run("zero limit means no eviction", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 0, time.Now) // 0 = unlimited

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		const n = 1_000

		for i := range n {
			_, err = store.NewRequest(t.Context(), someSessionID, fmt.Sprintf("r%d", i), storage.CapturedRequest{})
			assert.NoError(t, err)
		}

		seq, err := store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, err)
		assert.Seq2Count(t, n, seq)
	})
}

func testGetRequest(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("returns correct Data and Meta", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
			req   = storage.CapturedRequest{
				ClientAddr: "1.2.3.4",
				Method:     "POST",
				Body:       []byte(`{"key":"value"}`),
				Headers:    []storage.RequestHeader{{Name: "Content-Type", Value: "application/json"}},
				URL:        "https://example.com/webhook",
			}
			now = ft.Now()
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", req)
		assert.NoError(t, err)

		got, err := store.GetRequest(t.Context(), someSessionID, "r1")
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.DeepEqual(t, req, got.Data)
		assert.True(t, got.Meta.CreatedAt.Equal(now))
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		got, err := factory(t, 10, time.Now).GetRequest(t.Context(), "nonexistent", "r1")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("unknown request returns ErrRequestNotFound", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		got, err := store.GetRequest(t.Context(), someSessionID, "nonexistent")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, storage.ErrRequestNotFound)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returned request is a deep copy", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{Body: []byte("original")})
		assert.NoError(t, err)

		got, err := store.GetRequest(t.Context(), someSessionID, "r1")
		assert.NoError(t, err)

		got.Data.Body[0] = 'X' // mutate the returned copy

		// a subsequent read must still return the original data
		got2, err := store.GetRequest(t.Context(), someSessionID, "r1")
		assert.NoError(t, err)
		assert.DeepEqual(t, []byte("original"), got2.Data.Body)
	})
}

func testGetRequests(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("empty iterator for session with no requests", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		var iterErr error

		seq, err := store.GetRequests(t.Context(), someSessionID, &iterErr)
		assert.NoError(t, err)
		assert.Seq2Count(t, 0, seq)
		assert.NoError(t, iterErr)
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		seq, err := factory(t, 10, time.Now).GetRequests(t.Context(), "nonexistent", nil)
		assert.Nil(t, seq)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("requests yielded newest-first", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		for _, id := range []string{"r1", "r2", "r3"} {
			_, err = store.NewRequest(t.Context(), someSessionID, id, storage.CapturedRequest{})
			assert.NoError(t, err)

			ft.Advance(time.Second) // ensure strictly increasing CreatedAt
		}

		var iterErr error

		seq, err := store.GetRequests(t.Context(), someSessionID, &iterErr)
		assert.NoError(t, err)

		var ids []string

		for id := range seq {
			assert.NoError(t, iterErr)

			ids = append(ids, id)
		}

		assert.NoError(t, iterErr)
		assert.DeepEqual(t, []string{"r3", "r2", "r1"}, ids) // the order must be newest-first
	})

	t.Run("early break stops iteration", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		for _, id := range []string{"r1", "r2", "r3"} {
			_, err = store.NewRequest(t.Context(), someSessionID, id, storage.CapturedRequest{})
			assert.NoError(t, err)
			ft.Advance(time.Second)
		}

		seq, err := store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, err)

		var count int

		for range seq {
			count++
			break
		}

		assert.Equal(t, 1, count)
	})

	t.Run("large result set preserves newest-first order across batch boundaries", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 0, ft.Now) // 0 = unlimited, no eviction
		)

		const n = 1_001 // deliberately > 1000 to exercise second-batch fetches in paging implementations

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		want := make([]string, 0, n)

		for i := range n {
			rID := fmt.Sprintf("r%d", i)
			_, err = store.NewRequest(t.Context(), someSessionID, rID, storage.CapturedRequest{})
			assert.NoError(t, err)

			ft.Advance(time.Millisecond)

			want = append([]string{rID}, want...) // prepend
		}

		var iterErr error

		seq, err := store.GetRequests(t.Context(), someSessionID, &iterErr)
		assert.NoError(t, err)

		var got []string

		for id := range seq {
			assert.NoError(t, iterErr)

			got = append(got, id)
		}

		assert.NoError(t, iterErr)
		assert.Equal(t, n, len(got))
		assert.DeepEqual(t, want, got)
	})
}

func testDeleteRequest(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("deleted request is no longer accessible", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		assert.NoError(t, store.DeleteRequest(t.Context(), someSessionID, "r1"))

		_, err = store.GetRequest(t.Context(), someSessionID, "r1")
		assert.ErrorIs(t, err, storage.ErrRequestNotFound)
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(
			t,
			factory(t, 10, time.Now).DeleteRequest(t.Context(), "nonexistent", "r1"),
			storage.ErrSessionNotFound,
		)
	})

	t.Run("unknown request returns ErrRequestNotFound", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		assert.ErrorIs(
			t,
			store.DeleteRequest(t.Context(), someSessionID, "nonexistent"),
			storage.ErrRequestNotFound,
		)
	})
}

func testDeleteAllRequests(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("clears requests but leaves the session intact", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r2", storage.CapturedRequest{})
		assert.NoError(t, err)

		assert.NoError(t, store.DeleteAllRequests(t.Context(), someSessionID))

		seq, err := store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, err)
		assert.Seq2Count(t, 0, seq)

		_, err = store.GetSession(t.Context(), someSessionID)
		assert.NoError(t, err) // session itself is intact
	})

	t.Run("no-op for session with no requests", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		seq, err := store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, err)
		assert.Seq2Count(t, 0, seq)

		assert.NoError(t, store.DeleteAllRequests(t.Context(), someSessionID))

		seq, err = store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, err)
		assert.Seq2Count(t, 0, seq)
	})

	t.Run("session remains writable after clearing requests", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 10, time.Now)

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		assert.NoError(t, store.DeleteAllRequests(t.Context(), someSessionID))

		_, err = store.NewRequest(t.Context(), someSessionID, "r2", storage.CapturedRequest{})
		assert.NoError(t, err)

		seq, seqErr := store.GetRequests(t.Context(), someSessionID, nil)
		assert.NoError(t, seqErr)
		assert.Seq2Count(t, 1, seq)
	})

	t.Run("unknown session returns ErrSessionNotFound", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(
			t,
			factory(t, 10, time.Now).DeleteAllRequests(t.Context(), "nonexistent"),
			storage.ErrSessionNotFound,
		)
	})
}

func testSessionExpiry(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("all operations treat an expired session as not found", func(t *testing.T) {
		t.Parallel()

		var (
			ft    = newFakeTime()
			store = factory(t, 10, ft.Now)
		)

		const ttl = time.Minute

		_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, ttl)
		assert.NoError(t, err)

		_, err = store.NewRequest(t.Context(), someSessionID, "r1", storage.CapturedRequest{})
		assert.NoError(t, err)

		ft.Advance(ttl + time.Millisecond)

		// session-level operations
		_, err = store.GetSession(t.Context(), someSessionID)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)

		assert.ErrorIs(t, store.AddSessionTTL(t.Context(), someSessionID, time.Hour), storage.ErrSessionNotFound)
		assert.ErrorIs(t, store.DeleteSession(t.Context(), someSessionID), storage.ErrSessionNotFound)

		// request-level operations
		_, err = store.NewRequest(t.Context(), someSessionID, "r2", storage.CapturedRequest{})
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)

		_, err = store.GetRequest(t.Context(), someSessionID, "r1")
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)

		_, err = store.GetRequests(t.Context(), someSessionID, nil)
		assert.ErrorIs(t, err, storage.ErrSessionNotFound)

		assert.ErrorIs(t, store.DeleteRequest(t.Context(), someSessionID, "r1"), storage.ErrSessionNotFound)
		assert.ErrorIs(t, store.DeleteAllRequests(t.Context(), someSessionID), storage.ErrSessionNotFound)
	})
}

func testRaceProvocation(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("concurrent session and request operations", func(t *testing.T) {
		t.Parallel()

		store := factory(t, 1000, time.Now)

		var wg sync.WaitGroup

		for i := range 20 {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				sID := fmt.Sprintf("session-%d", i)

				_, _ = store.NewSession(t.Context(), sID, storage.SessionResponse{Code: 200}, storage.NoExpiration)
				_, _ = store.GetSession(t.Context(), sID)

				for j := range 20 {
					rID := fmt.Sprintf("req-%d-%d", i, j)
					_, _ = store.NewRequest(t.Context(), sID, rID, storage.CapturedRequest{Method: "POST"})
					_, _ = store.GetRequest(t.Context(), sID, rID)
				}

				rseq, _ := store.GetRequests(t.Context(), sID, nil)
				for range rseq {
				}

				_ = store.DeleteRequest(t.Context(), sID, fmt.Sprintf("req-%d-%d", i, 19))
				_ = store.DeleteAllRequests(t.Context(), sID)

				wg.Go(func() {
					_ = store.AddSessionTTL(t.Context(), sID, time.Hour)
				})

				_ = store.DeleteSession(t.Context(), sID)
			}(i)
		}

		wg.Wait()
	})
}

func testClose(t *testing.T, factory StorageFactory) {
	t.Helper()

	store := factory(t, 10, time.Now)

	closer, ok := store.(io.Closer)
	assert.True(t, ok) // must implement io.Closer for RunCloserSuite to be meaningful

	if !ok {
		t.Skip("storage does not implement io.Closer")

		return
	}

	assert.NoError(t, closer.Close())

	for name, fn := range map[string]func() error{
		"NewSession": func() error {
			_, err := store.NewSession(t.Context(), "s", storage.SessionResponse{}, time.Minute)
			return err
		},
		"GetSession":        func() error { _, err := store.GetSession(t.Context(), "s"); return err },
		"AddSessionTTL":     func() error { return store.AddSessionTTL(t.Context(), "s", time.Hour) },
		"DeleteSession":     func() error { return store.DeleteSession(t.Context(), "s") },
		"NewRequest":        func() error { _, err := store.NewRequest(t.Context(), "s", "r", storage.CapturedRequest{}); return err },
		"GetRequest":        func() error { _, err := store.GetRequest(t.Context(), "s", "r"); return err },
		"GetRequests":       func() error { _, err := store.GetRequests(t.Context(), "s", nil); return err },
		"DeleteRequest":     func() error { return store.DeleteRequest(t.Context(), "s", "r") },
		"DeleteAllRequests": func() error { return store.DeleteAllRequests(t.Context(), "s") },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.ErrorIs(t, fn(), storage.ErrClosed)
		})
	}
}

// RunPersistenceSuite exercises behaviors that require restarting storage with changed settings.
// The factory must return instances backed by the same underlying data on every call.
// Do not call this from in-memory implementations - persistence is meaningless there.
func RunPersistenceSuite(t *testing.T, factory StorageFactory) {
	t.Helper()
	t.Run("RequestLimitReduction", func(t *testing.T) { t.Parallel(); testRequestLimitReduction(t, factory) })
}

func testRequestLimitReduction(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("excess requests are evicted on the next write after reopening with a lower limit", func(t *testing.T) {
		t.Parallel()

		ft := newFakeTime()

		{ // phase 1: limit=5, add 6 requests; the oldest is evicted on the 6th add, leaving 5
			store := factory(t, 5, ft.Now)

			_, err := store.NewSession(t.Context(), someSessionID, storage.SessionResponse{}, storage.NoExpiration)
			assert.NoError(t, err)

			for i := range 6 {
				_, err = store.NewRequest(t.Context(), someSessionID, fmt.Sprintf("r%d", i), storage.CapturedRequest{})
				assert.NoError(t, err)
				ft.Advance(time.Second)
			}

			seq, seqErr := store.GetRequests(t.Context(), someSessionID, nil)
			assert.NoError(t, seqErr)
			assert.Seq2Count(t, 5, seq)

			if closer, ok := store.(io.Closer); ok {
				assert.NoError(t, closer.Close())
			}
		}

		{ // phase 2: reopen with limit=3; on the first NewRequest all excess entries are evicted at once
			store := factory(t, 3, ft.Now)

			_, err := store.NewRequest(t.Context(), someSessionID, "r6", storage.CapturedRequest{})
			assert.NoError(t, err)
			ft.Advance(time.Second)

			seq, seqErr := store.GetRequests(t.Context(), someSessionID, nil)
			assert.NoError(t, seqErr)
			assert.Seq2Count(t, 3, seq) // (5 excess - 3) + 1 new = 3

			_, err = store.NewRequest(t.Context(), someSessionID, "r7", storage.CapturedRequest{})
			assert.NoError(t, err)

			seq, seqErr = store.GetRequests(t.Context(), someSessionID, nil)
			assert.NoError(t, seqErr)
			assert.Seq2Count(t, 3, seq) // capped at 3
		}
	})
}

func testContextCancellationClosesStorage(t *testing.T, newStorage func(ctx context.Context) storage.Storage) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	store := newStorage(ctx)

	_, err := store.NewSession(ctx, "s1", storage.SessionResponse{}, storage.NoExpiration)
	assert.NoError(t, err)

	cancel()

	if closer, ok := store.(io.Closer); ok {
		_ = closer.Close() // blocks until the background goroutine exits
	}

	_, err = store.GetSession(t.Context(), "s1")
	assert.ErrorIs(t, err, storage.ErrClosed)
}

// --------------------------------------------------------------------------------------------------------------------

// RunBenchmarks covers the full [Storage] contract.
func RunBenchmarks(b *testing.B, factory StorageFactory) {
	b.Helper()
	b.ReportAllocs()

	const mainSID = "main-session"

	capturedReq := storage.CapturedRequest{
		ClientAddr: "10.0.0.1:54321",
		Method:     "POST",
		Headers:    []storage.RequestHeader{{Name: "Content-Type", Value: "application/json"}},
		Body:       []byte(`{"event":"push","repository":{"id":42}}`),
		URL:        "https://wh.example.com/" + mainSID,
	}
	sessionResp := storage.SessionResponse{
		Code:    200,
		Headers: []storage.ResponseHeader{{Name: "Content-Type", Value: "application/json"}},
	}

	b.Run("write", func(b *testing.B) {
		store := factory(b, 100, time.Now)
		ctx := b.Context()
		_, _ = store.NewSession(ctx, mainSID, sessionResp, storage.NoExpiration)

		b.ResetTimer()

		for i := range b.N {
			_, _ = store.NewRequest(ctx, mainSID, fmt.Sprintf("req-%d", i), capturedReq)
			if i%10 == 0 {
				seq, _ := store.GetRequests(ctx, mainSID, nil)
				for range seq {
				}
			}
		}
	})

	b.Run("read", func(b *testing.B) {
		const prefill = 50

		store := factory(b, 100, time.Now)
		ctx := b.Context()
		_, _ = store.NewSession(ctx, mainSID, sessionResp, storage.NoExpiration)

		for i := range prefill {
			_, _ = store.NewRequest(ctx, mainSID, fmt.Sprintf("req-%d", i), capturedReq)
		}

		b.ResetTimer()

		for i := range b.N {
			_, _ = store.GetRequest(ctx, mainSID, fmt.Sprintf("req-%d", i%prefill))
			if i%5 == 0 {
				seq, _ := store.GetRequests(ctx, mainSID, nil)
				for range seq {
				}
			}

			if i%7 == 0 {
				_, _ = store.GetSession(ctx, mainSID)
			}
		}
	})

	b.Run("manage", func(b *testing.B) {
		store := factory(b, 100, time.Now)
		ctx := b.Context()
		_, _ = store.NewSession(ctx, mainSID, sessionResp, storage.NoExpiration)

		b.ResetTimer()

		for i := range b.N {
			rID := fmt.Sprintf("req-%d", i)

			_, _ = store.NewRequest(ctx, mainSID, rID, capturedReq)
			if i%5 == 0 {
				_ = store.AddSessionTTL(ctx, mainSID, time.Hour)
			}

			if i%3 == 0 {
				_ = store.DeleteRequest(ctx, mainSID, rID)
			}

			if i%20 == 0 && i > 0 {
				_ = store.DeleteAllRequests(ctx, mainSID)
			}
		}
	})

	b.Run("lifecycle", func(b *testing.B) {
		store := factory(b, 100, time.Now)
		ctx := b.Context()
		b.ResetTimer()

		for i := range b.N {
			sid := fmt.Sprintf("session-%d", i)
			_, _ = store.NewSession(ctx, sid, sessionResp, time.Hour)
			_, _ = store.NewRequest(ctx, sid, "r0", capturedReq)
			_, _ = store.GetSession(ctx, sid)

			seq, _ := store.GetRequests(ctx, sid, nil)
			for range seq {
			}

			_ = store.DeleteSession(ctx, sid)
		}
	})
}

// --------------------------------------------------------------------------------------------------------------------

// fakeTime is a thread-safe, manually-advanceable clock.
// Each test case that needs time control should create its own fakeTime via newFakeTime so that
// parallel sub-tests do not interfere with each other's time progression.
type fakeTime struct{ v atomic.Pointer[time.Time] }

func newFakeTime() *fakeTime {
	ft := &fakeTime{}
	ft.v.Store(new(time.Now().Truncate(time.Second)))

	return ft
}

func (ft *fakeTime) Now() time.Time { return *ft.v.Load() }

// Advance moves the clock forward by the specified duration.
func (ft *fakeTime) Advance(d time.Duration) { ft.v.Store(new(ft.v.Load().Add(d))) }

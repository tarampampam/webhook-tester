package storage_test

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func toCloser(s storage.Storage) io.Closer {
	if c, ok := s.(io.Closer); ok {
		return c
	}

	return io.NopCloser(nil)
}

type fakeTime struct{ atomic.Pointer[time.Time] }

func (f *fakeTime) Add(t time.Duration) { newNow := f.Load().Add(t); f.Store(&newNow) }
func (f *fakeTime) Get() time.Time      { return *f.Load() }

func newFakeTime(t *testing.T) *fakeTime {
	t.Helper()

	now, ft := time.Now(), fakeTime{}
	ft.Store(&now)

	return &ft
}

func testSessionCreateReadDelete(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storage.Storage,
	sleep func(time.Duration),
) {
	t.Helper()

	var ctx = context.Background()

	t.Run("create, read, delete", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		var sessionHeaders = []storage.HttpHeader{{"foo", "bar"}, {"bar", "baz"}}

		const (
			code  uint16 = 201
			delay        = time.Second * 123
		)

		// create
		var sID, newErr = impl.NewSession(ctx, storage.Session{
			Code:    code,
			Headers: sessionHeaders,
			Delay:   delay,
		})

		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		// read
		got, getErr := impl.GetSession(ctx, sID)
		require.NoError(t, getErr)
		require.Equal(t, code, got.Code)
		require.Equal(t, sessionHeaders, got.Headers)
		require.Equal(t, delay, got.Delay)
		assert.NotZero(t, got.CreatedAtUnixMilli)

		// delete
		require.NoError(t, impl.DeleteSession(ctx, sID))                      // success
		require.ErrorIs(t, impl.DeleteSession(ctx, sID), storage.ErrNotFound) // already deleted
		require.ErrorIs(t, impl.DeleteSession(ctx, sID), storage.ErrSessionNotFound)

		// read again
		got, getErr = impl.GetSession(ctx, sID)
		require.Nil(t, got)
		require.ErrorIs(t, getErr, storage.ErrNotFound)
		require.ErrorIs(t, getErr, storage.ErrSessionNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		got, err := impl.GetSession(ctx, "foo")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete not existing", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		require.ErrorIs(t, impl.DeleteSession(ctx, "foo"), storage.ErrSessionNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()

		const sessionTTL = time.Millisecond

		var impl = new(sessionTTL, 1)
		defer func() { _ = toCloser(impl).Close() }()

		sID, err := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		sleep(sessionTTL * 2) // wait for expiration

		_, err = impl.GetSession(ctx, sID)

		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("add session TTL", func(t *testing.T) {
		t.Parallel()

		const sessionTTL = time.Millisecond * 20

		var impl = new(sessionTTL, 2)
		defer func() { _ = toCloser(impl).Close() }()

		var now = time.Now()

		// create session with TTL
		sID, err := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		// get it (ensure it exists)
		sess, err := impl.GetSession(ctx, sID)
		require.NoError(t, err)

		{ // check the created and expiration time
			require.InDelta(t, now.UnixMilli(), sess.CreatedAtUnixMilli, 50)
			require.InDelta(t, now.Add(sessionTTL).UnixMilli(), sess.ExpiresAt.UnixMilli(), 40)
		}

		var ( // store the original values
			originalCreatedAt = sess.CreatedAtUnixMilli
			originalExpiresAt = sess.ExpiresAt
		)

		// reload the session
		sess, err = impl.GetSession(ctx, sID)
		require.NoError(t, err)
		require.Equal(t, originalCreatedAt, sess.CreatedAtUnixMilli) // should be the same
		require.InDelta(t, originalExpiresAt.UnixMilli(), sess.ExpiresAt.UnixMilli(), 10)

		// add TTL
		require.NoError(t, impl.AddSessionTTL(ctx, sID, sessionTTL*2)) // current ttl = x + 2x = 3x

		// wait for expiration (2x)
		sleep(sessionTTL * 2)

		// the session should be still alive
		sess, err = impl.GetSession(ctx, sID)
		require.NoError(t, err)
		require.Equal(t, originalCreatedAt, sess.CreatedAtUnixMilli)
		require.NotEqual(t, originalExpiresAt, sess.ExpiresAt) // changed

		// wait for expiration (2x)
		sleep(sessionTTL * 2)

		// check again
		sess, err = impl.GetSession(ctx, sID)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
		require.Nil(t, sess)
	})
}

func testRequestCreateReadDelete(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storage.Storage,
	sleep func(time.Duration),
) {
	t.Helper()

	var ctx = context.Background()

	const someUrl = "https://example.com/foo/bar"

	t.Run("create, read, delete", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, newErr := impl.NewSession(ctx, storage.Session{
			Code:    201,
			Headers: []storage.HttpHeader{{"foo", "bar"}, {"bar", "baz"}},
			Delay:   time.Second,
		})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		const (
			clientAddr = "127.0.0.1"
			method     = "GET"
			body       = " \nfoo bar\n\t \nbaz"
		)

		var requestHeaders = []storage.HttpHeader{{"foo", "bar"}, {"bar", "baz"}}

		// create
		rID, newReqErr := impl.NewRequest(ctx, sID, storage.Request{
			ClientAddr: clientAddr,
			Method:     method,
			Body:       []byte(body),
			Headers:    requestHeaders,
			URL:        someUrl,
		})
		require.NoError(t, newReqErr)
		require.NotEmpty(t, rID)

		// read
		got, getErr := impl.GetRequest(ctx, sID, rID)
		require.NoError(t, getErr)
		require.Equal(t, clientAddr, got.ClientAddr)
		require.Equal(t, method, got.Method)
		require.Equal(t, []byte(body), got.Body)
		require.Equal(t, requestHeaders, got.Headers)
		require.Equal(t, someUrl, got.URL)
		assert.NotZero(t, got.CreatedAtUnixMilli)

		{ // read all
			all, err := impl.GetAllRequests(ctx, sID)
			require.NoError(t, err)
			require.Len(t, all, 1)
			require.Equal(t, all, map[string]storage.Request{rID: *got})
		}

		// delete
		require.NoError(t, impl.DeleteRequest(ctx, sID, rID))                      // success
		require.ErrorIs(t, impl.DeleteRequest(ctx, sID, rID), storage.ErrNotFound) // already deleted
		require.ErrorIs(t, impl.DeleteRequest(ctx, sID, rID), storage.ErrRequestNotFound)

		// read again
		got, getErr = impl.GetRequest(ctx, sID, rID)
		require.Nil(t, got)
		require.ErrorIs(t, getErr, storage.ErrNotFound)
		require.ErrorIs(t, getErr, storage.ErrRequestNotFound)
	})

	t.Run("new request - limit exceeded", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 2) // limit is 2
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, err := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		// create request #1
		rID1, err := impl.NewRequest(ctx, sID, storage.Request{ClientAddr: "req1"})
		require.NoError(t, err)
		require.NotEmpty(t, rID1)

		sleep(time.Millisecond) // the accuracy is one millisecond

		// create request #2
		rID2, err := impl.NewRequest(ctx, sID, storage.Request{ClientAddr: "req2"})
		require.NoError(t, err)
		require.NotEmpty(t, rID2)

		// now, the session has 2 requests and the limit is reached

		{ // check made requests
			requests, _ := impl.GetAllRequests(ctx, sID)
			require.Len(t, requests, 2)
			_, ok := requests[rID1]
			require.True(t, ok)
			_, ok = requests[rID2]
			require.True(t, ok)

			req, _ := impl.GetRequest(ctx, sID, rID1)
			require.NotNil(t, req)

			req, _ = impl.GetRequest(ctx, sID, rID2)
			require.NotNil(t, req)
		}

		sleep(time.Millisecond)

		// create request #3
		rID3, err := impl.NewRequest(ctx, sID, storage.Request{ClientAddr: "req3"})
		require.NoError(t, err)
		require.NotEmpty(t, rID3)

		// now, the request #1 should be deleted because the limit is reached (the storage should keep the requests
		// with numbers 2 and 3)

		{ // check made requests again
			requests, _ := impl.GetAllRequests(ctx, sID)
			require.Len(t, requests, 2) // still 2
			_, ok := requests[rID2]
			require.True(t, ok)
			_, ok = requests[rID3]
			require.True(t, ok)

			req, reqErr := impl.GetRequest(ctx, sID, rID1) // not found
			require.Nil(t, req)
			require.Error(t, reqErr)

			req, _ = impl.GetRequest(ctx, sID, rID2) // ok
			require.NotNil(t, req)

			req, _ = impl.GetRequest(ctx, sID, rID3) // ok
			require.NotNil(t, req)
		}

		// and now add one more request - after that, the request #2 should be deleted (the storage should keep the
		// requests with numbers 3 and 4)

		sleep(time.Millisecond)

		// create request #4
		rID4, err := impl.NewRequest(ctx, sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID4)

		{ // check made requests again
			requests, _ := impl.GetAllRequests(ctx, sID)
			require.Len(t, requests, 2) // still 2

			req, reqErr := impl.GetRequest(ctx, sID, rID1) // not found
			require.Nil(t, req)
			require.Error(t, reqErr)

			req, reqErr = impl.GetRequest(ctx, sID, rID2) // not found
			require.Nil(t, req)
			require.Error(t, reqErr)

			req, _ = impl.GetRequest(ctx, sID, rID3) // ok
			require.NotNil(t, req)

			req, _ = impl.GetRequest(ctx, sID, rID4) // ok
			require.NotNil(t, req)
		}

		// and now delete all the requests
		require.NoError(t, impl.DeleteAllRequests(ctx, sID))

		_, err = impl.GetAllRequests(ctx, sID)
		require.NoError(t, err)

		// and the session
		require.NoError(t, impl.DeleteSession(ctx, sID))

		_, err = impl.GetAllRequests(ctx, sID)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete all", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, err := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		// create request
		rID, err := impl.NewRequest(ctx, sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID)

		// delete all
		require.NoError(t, impl.DeleteAllRequests(ctx, sID))

		// check
		all, err := impl.GetAllRequests(ctx, sID)
		require.NoError(t, err)
		require.Empty(t, all)
	})

	t.Run("delete all - no session", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		err := impl.DeleteAllRequests(ctx, "foo")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get all - empty", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, err := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		all, err := impl.GetAllRequests(ctx, sID)
		require.NoError(t, err)
		require.Empty(t, all)
	})

	t.Run("get all - no session", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		all, err := impl.GetAllRequests(ctx, "foo")
		require.Nil(t, all)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("new request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		_, err := impl.NewRequest(ctx, "foo", storage.Request{})
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		got, err := impl.GetRequest(ctx, "foo", "bar")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get request - request not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, newErr := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		got, err := impl.GetRequest(ctx, sID, "foo")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrRequestNotFound)
	})

	t.Run("delete request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		err := impl.DeleteRequest(ctx, "foo", "bar")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete request - request not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = toCloser(impl).Close() }()

		// create session
		sID, newErr := impl.NewSession(ctx, storage.Session{})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		err := impl.DeleteRequest(ctx, sID, "foo")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrRequestNotFound)
	})
}

func testRaceProvocation(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storage.Storage,
) {
	t.Helper()

	var ctx = context.Background()

	var impl = new(time.Minute, 1000)
	defer func() { _ = toCloser(impl).Close() }()

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			sID, err := impl.NewSession(ctx, storage.Session{})
			require.NoError(t, err)

			_, err = impl.GetSession(ctx, sID)
			require.NoError(t, err)

			var rID string

			for range 20 {
				rID, err = impl.NewRequest(ctx, sID, storage.Request{})
				require.NoError(t, err)

				_, err = impl.GetRequest(ctx, sID, rID)
				require.NoError(t, err)

				all, aErr := impl.GetAllRequests(ctx, sID)
				require.NoError(t, aErr)
				require.NotEmpty(t, all)
			}

			require.NoError(t, impl.AddSessionTTL(ctx, sID, time.Minute))

			require.NoError(t, impl.DeleteRequest(ctx, sID, rID))

			require.NoError(t, impl.DeleteAllRequests(ctx, sID))
		}()
	}

	wg.Wait()
}

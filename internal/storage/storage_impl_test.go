package storage_test

import (
	"io"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/internal/storage"
)

type storageToTest interface {
	storage.Storage
	io.Closer
}

func testSessionCreateReadDelete(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storageToTest,
) {
	t.Helper()

	t.Run("create, read, delete", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		const (
			code        uint16 = 201
			content            = " \nfoo bar\n\t \nbaz"
			contentType        = "text/javascript"
			delay              = time.Second * 123
		)

		// create
		var sID, newErr = impl.NewSession(storage.Session{
			Code:        code,
			Content:     []byte(content),
			ContentType: contentType,
			Delay:       delay,
		})

		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		// read
		got, getErr := impl.GetSession(sID)
		require.NoError(t, getErr)
		require.Equal(t, code, got.Code)
		require.Equal(t, []byte(content), got.Content)
		require.Equal(t, contentType, got.ContentType)
		require.Equal(t, delay, got.Delay)
		assert.NotZero(t, got.CreatedAt)

		// delete
		require.NoError(t, impl.DeleteSession(sID))                      // success
		require.ErrorIs(t, impl.DeleteSession(sID), storage.ErrNotFound) // already deleted
		require.ErrorIs(t, impl.DeleteSession(sID), storage.ErrSessionNotFound)

		// read again
		got, getErr = impl.GetSession(sID)
		require.Nil(t, got)
		require.ErrorIs(t, getErr, storage.ErrNotFound)
		require.ErrorIs(t, getErr, storage.ErrSessionNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		got, err := impl.GetSession("foo")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete not existing", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		require.ErrorIs(t, impl.DeleteSession("foo"), storage.ErrSessionNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()

		const sessionTTL = time.Millisecond

		var impl = new(sessionTTL, 1)
		defer func() { _ = impl.Close() }()

		sID, err := impl.NewSession(storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		<-time.After(sessionTTL * 2) // wait for expiration

		_, err = impl.GetSession(sID)

		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("closed", func(t *testing.T) {
		t.Parallel()

		impl := new(time.Minute, 1)
		require.NoError(t, impl.Close())
		require.ErrorIs(t, impl.Close(), storage.ErrClosed) // second close

		_, err := impl.NewSession(storage.Session{})
		require.ErrorIs(t, err, storage.ErrClosed)

		_, err = impl.GetSession("foo")
		require.ErrorIs(t, err, storage.ErrClosed)

		err = impl.DeleteSession("foo")
		require.ErrorIs(t, err, storage.ErrClosed)
	})
}

func testRequestCreateReadDelete(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storageToTest,
) {
	t.Helper()

	someUrl, _ := url.Parse("https://example.com/foo/bar")

	t.Run("create, read, delete", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		// create session
		sID, newErr := impl.NewSession(storage.Session{
			Code:        201,
			Content:     []byte("foo bar"),
			ContentType: "text/javascript",
			Delay:       time.Second,
		})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		const (
			clientAddr = "127.0.0.1"
			method     = "GET"
			body       = " \nfoo bar\n\t \nbaz"
		)

		var (
			headers = map[string]string{"foo": "bar", "bar": "baz"}
		)

		// create
		rID, newReqErr := impl.NewRequest(sID, storage.Request{
			ClientAddr: clientAddr,
			Method:     method,
			Body:       []byte(body),
			Headers:    headers,
			URL:        *someUrl,
		})
		require.NoError(t, newReqErr)
		require.NotEmpty(t, rID)

		// read
		got, getErr := impl.GetRequest(sID, rID)
		require.NoError(t, getErr)
		require.Equal(t, clientAddr, got.ClientAddr)
		require.Equal(t, method, got.Method)
		require.Equal(t, []byte(body), got.Body)
		require.Equal(t, headers, got.Headers)
		require.Equal(t, *someUrl, got.URL)
		assert.NotZero(t, got.CreatedAt)

		{ // read all
			all, err := impl.GetAllRequests(sID)
			require.NoError(t, err)
			require.Len(t, all, 1)
			require.Equal(t, all, map[string]storage.Request{rID: *got})
		}

		// delete
		require.NoError(t, impl.DeleteRequest(sID, rID))                      // success
		require.ErrorIs(t, impl.DeleteRequest(sID, rID), storage.ErrNotFound) // already deleted
		require.ErrorIs(t, impl.DeleteRequest(sID, rID), storage.ErrRequestNotFound)

		// read again
		got, getErr = impl.GetRequest(sID, rID)
		require.Nil(t, got)
		require.ErrorIs(t, getErr, storage.ErrNotFound)
		require.ErrorIs(t, getErr, storage.ErrRequestNotFound)
	})

	t.Run("new request - limit exceeded", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 2) // limit is 2
		defer func() { _ = impl.Close() }()

		// create session
		sID, err := impl.NewSession(storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		// create request #1
		rID1, err := impl.NewRequest(sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID1)

		// create request #2
		rID2, err := impl.NewRequest(sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID2)

		{ // check made requests
			requests, _ := impl.GetAllRequests(sID)
			require.Len(t, requests, 2)

			req, _ := impl.GetRequest(sID, rID1)
			require.NotNil(t, req)

			req, _ = impl.GetRequest(sID, rID2)
			require.NotNil(t, req)
		}

		// create request #3
		rID3, err := impl.NewRequest(sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID3)

		{ // check made requests again
			requests, _ := impl.GetAllRequests(sID)
			require.Len(t, requests, 2) // still 2

			req, reqErr := impl.GetRequest(sID, rID1) // not found
			require.Nil(t, req)
			require.Error(t, reqErr)

			req, _ = impl.GetRequest(sID, rID2) // ok
			require.NotNil(t, req)

			req, _ = impl.GetRequest(sID, rID3) // ok
			require.NotNil(t, req)
		}

		// create request #4
		rID4, err := impl.NewRequest(sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID4)

		{ // check made requests again
			requests, _ := impl.GetAllRequests(sID)
			require.Len(t, requests, 2) // still 2

			req, reqErr := impl.GetRequest(sID, rID2) // not found
			require.Nil(t, req)
			require.Error(t, reqErr)

			req, _ = impl.GetRequest(sID, rID3) // ok
			require.NotNil(t, req)

			req, _ = impl.GetRequest(sID, rID4) // ok
			require.NotNil(t, req)
		}

		// and now delete all the requests
		require.NoError(t, impl.DeleteAllRequests(sID))

		_, err = impl.GetAllRequests(sID)
		require.NoError(t, err)

		// and the session
		require.NoError(t, impl.DeleteSession(sID))

		_, err = impl.GetAllRequests(sID)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete all", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		// create session
		sID, err := impl.NewSession(storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		// create request
		rID, err := impl.NewRequest(sID, storage.Request{})
		require.NoError(t, err)
		require.NotEmpty(t, rID)

		// delete all
		require.NoError(t, impl.DeleteAllRequests(sID))

		// check
		all, err := impl.GetAllRequests(sID)
		require.NoError(t, err)
		require.Empty(t, all)
	})

	t.Run("delete all - no session", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		err := impl.DeleteAllRequests("foo")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get all - empty", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		// create session
		sID, err := impl.NewSession(storage.Session{})
		require.NoError(t, err)
		require.NotEmpty(t, sID)

		all, err := impl.GetAllRequests(sID)
		require.NoError(t, err)
		require.Empty(t, all)
	})

	t.Run("get all - no session", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		all, err := impl.GetAllRequests("foo")
		require.Nil(t, all)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("new request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		_, err := impl.NewRequest("foo", storage.Request{})
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		got, err := impl.GetRequest("foo", "bar")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("get request - request not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		// create session
		sID, newErr := impl.NewSession(storage.Session{})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		got, err := impl.GetRequest(sID, "foo")
		require.Nil(t, got)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrRequestNotFound)
	})

	t.Run("delete request - session not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		err := impl.DeleteRequest("foo", "bar")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("delete request - request not found", func(t *testing.T) {
		t.Parallel()

		var impl = new(time.Minute, 1)
		defer func() { _ = impl.Close() }()

		// create session
		sID, newErr := impl.NewSession(storage.Session{})
		require.NoError(t, newErr)
		require.NotEmpty(t, sID)

		err := impl.DeleteRequest(sID, "foo")
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrRequestNotFound)
	})

	t.Run("closed", func(t *testing.T) {
		t.Parallel()

		impl := new(time.Minute, 1)
		require.NoError(t, impl.Close())
		require.ErrorIs(t, impl.Close(), storage.ErrClosed) // second close

		_, err := impl.NewRequest("foo", storage.Request{})
		require.ErrorIs(t, err, storage.ErrClosed)

		_, err = impl.GetRequest("foo", "bar")
		require.ErrorIs(t, err, storage.ErrClosed)

		_, err = impl.GetAllRequests("foo")
		require.ErrorIs(t, err, storage.ErrClosed)

		err = impl.DeleteRequest("foo", "bar")
		require.ErrorIs(t, err, storage.ErrClosed)

		err = impl.DeleteAllRequests("foo")
		require.ErrorIs(t, err, storage.ErrClosed)
	})
}

func testRaceProvocation(
	t *testing.T,
	new func(sessionTTL time.Duration, maxRequests uint32) storageToTest,
) {
	t.Helper()

	var impl = new(time.Minute, 1000)
	defer func() { _ = impl.Close() }()

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			sID, err := impl.NewSession(storage.Session{})
			require.NoError(t, err)

			_, err = impl.GetSession(sID)
			require.NoError(t, err)

			var rID string

			for range 50 {
				rID, err = impl.NewRequest(sID, storage.Request{})
				require.NoError(t, err)

				_, err = impl.GetRequest(sID, rID)
				require.NoError(t, err)

				all, aErr := impl.GetAllRequests(sID)
				require.NoError(t, aErr)
				require.NotEmpty(t, all)
			}

			require.NoError(t, impl.DeleteRequest(sID, rID))

			require.NoError(t, impl.DeleteAllRequests(sID))
		}()
	}

	wg.Wait()
}

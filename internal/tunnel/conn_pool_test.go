package tunnel_test

import (
	"context"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/tunnel"
)

type fakeDialer struct {
	dialCount atomic.Int32
	giveConn  net.Conn
	giveErr   error
}

func (fd *fakeDialer) dial(_ context.Context, _, _ string) (net.Conn, error) {
	fd.dialCount.Add(1)

	return fd.giveConn, fd.giveErr
}

func TestConnectionsPool_Get(t *testing.T) {
	t.Parallel()

	var (
		ctx    = context.Background()
		dialer = &fakeDialer{giveConn: &net.TCPConn{}}
	)

	// create a new connections pool
	tc, stop := tunnel.NewConnectionsPool(ctx, "", 10, tunnel.WithConnectionsPoolDialer(dialer.dial))

	t.Cleanup(stop) // schedule the pool to be stopped on test exit

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// get a new connection from the pool
			conn, got := tc.Get(ctx)
			require.True(t, got)
			require.NotNil(t, conn.Conn)
			require.NotPanics(t, conn.Release)
		}()
	}

	wg.Wait() // wait until all goroutines are finished

	<-time.After(time.Millisecond) // wait for the last dial to be completed
	runtime.Gosched()              // give the scheduler a chance to run

	// the dial function should be called 110 times (10 initial dials + 100 dials from goroutines)
	require.Equal(t, int32(100+10), dialer.dialCount.Load())

	stop() // after this line, each attempt to get a new connection should return false and nil connection

	for range 100 {
		conn, got := tc.Get(ctx)
		require.False(t, got)
		require.Nil(t, conn.Conn)
		require.NotPanics(t, conn.Release)
	}

	conn, got := tc.Get() // without context
	require.False(t, got)
	require.Nil(t, conn.Conn)
}

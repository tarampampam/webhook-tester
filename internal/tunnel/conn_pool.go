package tunnel

import (
	"context"
	"net"
	"sync"
)

type (
	ConnectionsPool struct {
		poolCh    chan Connection // a channel to keep active connections
		needNewCh chan struct{}   // a channel with notifications about the need to create a new connection
		stop      chan struct{}   // a channel to stop all goroutines
		onError   func(error)     // may be nil or a function to call on error
		dialer    func(_ context.Context, network, address string) (net.Conn, error)
	}

	Connection struct {
		Conn      net.Conn
		onRelease func() // a function to call when the connection is released
	}
)

// Release closes the connection and notifies the pool about the need to create a new connection.
// Call this method when you no longer need the connection, this is very important to keep the pool!
func (rc Connection) Release() {
	if rc.Conn != nil {
		_ = rc.Conn.Close()
	}

	if rc.onRelease != nil {
		rc.onRelease()
	}
}

// ConnectionsPoolOption allows you to configure the ConnectionsPool during creation.
type ConnectionsPoolOption func(*ConnectionsPool)

// WithConnectionsPoolErrorsHandler sets the error handler for the ConnectionsPool.
func WithConnectionsPoolErrorsHandler(fn func(error)) ConnectionsPoolOption {
	return func(cp *ConnectionsPool) { cp.onError = fn }
}

// WithConnectionsPoolDialer sets the dialer for the ConnectionsPool. By default, the pool uses the
// [net.Dialer.DialContext] method. Useful for testing purposes.
func WithConnectionsPoolDialer(dialer func(context.Context, string, string) (net.Conn, error)) ConnectionsPoolOption {
	return func(cp *ConnectionsPool) { cp.dialer = dialer }
}

// NewConnectionsPool creates a new pool of connections to the remote server. The pool will create
// connections in the background and will keep the total number of connections equal to the size
// parameter. The pool will create new connections when the connection will be released using the
// [Connection.Release] method.
//
// To close the pool and release all connections, call the returned cleanup function (it will close
// all connections, stop all goroutines and empty the pool of connections).
func NewConnectionsPool( //nolint:funlen
	ctx context.Context,
	remoteAddr string,
	size uint,
	opts ...ConnectionsPoolOption,
) (ConnectionsPool, func()) {
	var cp = ConnectionsPool{
		poolCh:    make(chan Connection, size),
		needNewCh: make(chan struct{}, size),
		stop:      make(chan struct{}),
		dialer:    (&net.Dialer{}).DialContext, // default dialer
	}

	for _, opt := range opts {
		opt(&cp)
	}

	var wg sync.WaitGroup // is used to wait for all goroutines to exit

	for range size {
		// fill the channel with the initial amount of notifications
		cp.needNewCh <- struct{}{}

		// increment the counter of the goroutines
		wg.Add(1)

		// the following goroutine is responsible for creating new connections to the remote server
		// and adding them to the pool. in case of an error, it will notify the pool about the need
		// to create a new connection. the goroutine will exit when the context is canceled or the
		// stop channel is closed
		go func() {
			var (
				conn    net.Conn
				connErr error
			)

			defer func() {
				if conn != nil { // close the last established connection on exit
					_ = conn.Close()
				}

				wg.Done()
			}()

			for { // infinite loop to create new connections
				select {
				case <-ctx.Done():
					return // exit the goroutine if the context was canceled
				case <-cp.stop:
					return // or if the stop channel was closed
				case _, isOpened := <-cp.needNewCh:
					if !isOpened { // if the channel is closed, exit the goroutine
						return
					}

					if ctx.Err() != nil { // check if the context was canceled
						return
					}

					// try to establish a connection to the remote server
					conn, connErr = cp.dialer(ctx, "tcp", remoteAddr)
					if connErr != nil { // on connection error
						if cp.onError != nil {
							cp.onError(connErr) // call the error handler
						}

						select {
						case <-ctx.Done():
							return
						case <-cp.stop:
							return
						case cp.needNewCh <- struct{}{}:
							continue // notify about the need to create a new connection, and jump to retry
						}
					}

					// on success, add the connection to the poolCh
					cp.poolCh <- Connection{
						Conn: conn,
						onRelease: func() {
							select {
							case <-ctx.Done():
							case <-cp.stop:
							case cp.needNewCh <- struct{}{}: // notify about the need to create a new connection
							}
						},
					}
				}
			}
		}()
	}

	return cp, sync.OnceFunc(func() {
		close(cp.stop) // close the stop channel to stop all goroutines
		wg.Wait()      // wait for all goroutines to exit

		// empty the poolCh
		for len(cp.poolCh) > 0 {
			conn := <-cp.poolCh
			conn.Release()
		}

		close(cp.poolCh) // close the poolCh channel

		// empty the needNewCh channel
		for len(cp.needNewCh) > 0 {
			<-cp.needNewCh
		}

		close(cp.needNewCh) // close the needNewCh channel
	})
}

// Get returns an active connection from the pool. If the pool is empty, the method will block until
// a new connection is established and added to the pool, or the context is canceled.
//
// Use the 2nd return value to check if the connection was successfully received.
func (cp ConnectionsPool) Get(optionalCtx ...context.Context) (Connection, bool) {
	var ctx context.Context

	if len(optionalCtx) > 0 {
		ctx = optionalCtx[0]
	} else {
		ctx = context.Background()
	}

	select {
	case <-cp.stop:
		return Connection{}, false
	case <-ctx.Done():
		return Connection{}, false
	case conn, isOpened := <-cp.poolCh:
		return conn, isOpened
	}
}

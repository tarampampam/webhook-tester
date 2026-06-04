package tunnel

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrAlreadyStarted is returned when trying to start a tunnel that is already running.
	ErrAlreadyStarted = errors.New("tunnel already started")
)

// Tunneler is the interface for managing an outbound tunnel.
type Tunneler interface {
	io.Closer

	// Expose starts a tunnel to the local port and returns the public URL.
	Expose(ctx context.Context, localPort uint16) (string, error)
}

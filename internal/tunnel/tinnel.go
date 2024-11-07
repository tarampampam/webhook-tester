package tunnel

import (
	"context"
)

type Tunneler interface {
	// Close the tunnel.
	Close() error

	// Expose starts a tunnel to the local port and returns the public URL. To close/stop the tunnel, call Close.
	Expose(ctx context.Context, localPort uint16) (string, error)
}

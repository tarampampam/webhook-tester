package tunnel

import (
	"context"
	"fmt"
)

var _ Tunneler = (*cloudflareQuickTunnel)(nil)

type cloudflareQuickTunnel struct {
	ctx   context.Context
	close func()
}

func (c *cloudflareQuickTunnel) Close() error {
	c.close()
	return nil
}

func (c *cloudflareQuickTunnel) Expose(ctx context.Context, localPort uint16) (string, error) {
	c.ctx = ctx
	url, tunnel, err := createQuickTunnel()
	if err != nil {
		return "", err
	}

	close, err := startCloudflareTunnel(fmt.Sprintf("http://localhost:%d", localPort), tunnel)
	if err != nil {
		return "", err
	}

	c.close = close
	return url, nil
}

func NewCloudflareQuickTunnel() Tunneler {
	return &cloudflareQuickTunnel{
		ctx:   context.Background(),
		close: func() {},
	}
}

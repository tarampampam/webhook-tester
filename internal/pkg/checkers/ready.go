package checkers

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// ReadyChecker is a readiness checker.
type ReadyChecker struct {
	ctx context.Context
	rdb *redis.Client // can be nil
}

// NewReadyChecker creates readiness checker.
func NewReadyChecker(ctx context.Context, rdb *redis.Client) *ReadyChecker {
	return &ReadyChecker{ctx: ctx, rdb: rdb}
}

// Check application is ready for incoming requests processing?
func (c *ReadyChecker) Check() error {
	if c.rdb != nil {
		if err := c.rdb.Ping(c.ctx).Err(); err != nil {
			return err
		}
	}

	return nil
}

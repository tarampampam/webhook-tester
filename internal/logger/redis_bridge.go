package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type redisBridge struct {
	zap *zap.Logger
}

// NewRedisBridge creates instance that can ba used as a bridge between zap and redis client for logging.
func NewRedisBridge(zap *zap.Logger) *redisBridge { return &redisBridge{zap: zap} } //nolint:golint

// Printf implements redis logger interface.
func (rb *redisBridge) Printf(_ context.Context, format string, v ...any) {
	rb.zap.Warn(fmt.Sprintf(format, v...), zap.String("source", "redis"))
}

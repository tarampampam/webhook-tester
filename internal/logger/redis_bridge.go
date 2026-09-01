package logger

import (
	"context"
	"fmt"
)

type redisBridge struct {
	log *Logger
	lvl Level
}

// NewRedisBridge creates instance that can ba used as a bridge between [Logger] and redis client for logging.
func NewRedisBridge(log *Logger, lvl Level) interface {
	Printf(ctx context.Context, format string, v ...any)
} {
	return &redisBridge{log: log, lvl: lvl}
}

func (br redisBridge) Printf(_ context.Context, format string, v ...any) {
	br.log.Log(br.lvl, fmt.Sprintf(format, v...))
}

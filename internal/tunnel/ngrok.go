package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	ngrokLog "golang.ngrok.com/ngrok/log"
)

type Ngrok struct {
	tunnel    atomic.Pointer[ngrok.Forwarder]
	authToken string
	log       ngrokLog.Logger
}

// NgrokOption is a functional option for the Ngrok instance.
type NgrokOption func(*Ngrok)

// WithNgrokLogger sets the logger for the Ngrok instance.
func WithNgrokLogger(log *zap.Logger) NgrokOption {
	return func(n *Ngrok) { n.log = &ngrokLogAdapter{zap: log} }
}

// NewNgrok creates a new Ngrok instance with the given auth token and options.
func NewNgrok(authToken string, opts ...NgrokOption) *Ngrok {
	var n = Ngrok{
		authToken: authToken,
		log:       &ngrokLogAdapter{zap: zap.NewNop()},
	}

	for _, opt := range opts {
		opt(&n)
	}

	return &n
}

func (n *Ngrok) Expose(ctx context.Context, localPort uint16) (string, error) {
	if n.tunnel.Load() != nil {
		return "", errors.New("tunnel already started")
	}

	var backendUrl, uErr = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", localPort))
	if uErr != nil {
		return "", fmt.Errorf("failed to parse backend url: %w", uErr)
	}

	ln, tErr := ngrok.ListenAndForward(
		ctx,
		backendUrl,
		config.HTTPEndpoint(),
		ngrok.WithAuthtoken(n.authToken),
		ngrok.WithLogger(n.log),
	)
	if tErr != nil {
		return "", tErr
	}

	n.tunnel.Store(&ln)

	return ln.URL(), nil
}

func (n *Ngrok) Close() error {
	if old := n.tunnel.Swap(nil); old != nil {
		return (*old).Close()
	}

	return errors.New("tunnel not started")
}

// ngrokLogAdapter is an adapter for the [ngrokLog.Logger] interface.
type ngrokLogAdapter struct{ zap *zap.Logger }

var _ ngrokLog.Logger = (*ngrokLogAdapter)(nil) // ensure ngrokLogAdapter implements [ngrokLog.Logger]

// Log a message at the given level with data key/value pairs. data may be nil.
func (n *ngrokLogAdapter) Log(_ context.Context, level ngrokLog.LogLevel, msg string, data map[string]any) {
	var lvl zapcore.Level

	switch level {
	case ngrokLog.LogLevelTrace:
		lvl = zapcore.DebugLevel
	case ngrokLog.LogLevelDebug:
		lvl = zapcore.DebugLevel
	case ngrokLog.LogLevelInfo:
		lvl = zapcore.InfoLevel
	case ngrokLog.LogLevelWarn:
		lvl = zapcore.WarnLevel
	case ngrokLog.LogLevelError:
		lvl = zapcore.ErrorLevel
	case ngrokLog.LogLevelNone:
		lvl = zapcore.DebugLevel
	default:
		n.zap.Error(fmt.Sprintf("invalid log level: %v", level))

		return
	}

	if ce := n.zap.Check(lvl, msg); ce != nil {
		var fields = make([]zap.Field, 0, len(data))

		for k, v := range data {
			fields = append(fields, zap.Any(k, v))
		}

		ce.Write(fields...)
	}
}

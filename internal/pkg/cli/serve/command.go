// Package serve contains CLI `serve` command implementation.
package serve

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cobra"
	"github.com/tarampampam/webhook-tester/internal/pkg/breaker"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	appHttp "github.com/tarampampam/webhook-tester/internal/pkg/http"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"go.uber.org/zap"
)

// broadcast driver names
const brDriverNone, brDriverPusher = "none", "pusher"

// NewCommand creates `serve` command.
func NewCommand(ctx context.Context, log *zap.Logger) *cobra.Command {
	var f flags

	cmd := &cobra.Command{
		Use:     "serve",
		Aliases: []string{"s", "server"},
		Short:   "Start HTTP server",
		Long:    "Environment variables have higher priority then flags",
		PreRunE: func(*cobra.Command, []string) error {
			if err := f.overrideUsingEnv(); err != nil {
				return err
			}

			return f.validate()
		},
		RunE: func(*cobra.Command, []string) error {
			return run(ctx, log, &f)
		},
	}

	f.init(cmd.Flags())

	return cmd
}

const serverShutdownTimeout = 5 * time.Second

// run current command.
func run(parentCtx context.Context, log *zap.Logger, f *flags) error { //nolint:funlen,gocyclo
	var (
		ctx, cancel = context.WithCancel(parentCtx) // serve context creation
		oss         = breaker.NewOSSignals(ctx)     // OS signals listener
	)

	// subscribe for system signals
	oss.Subscribe(func(sig os.Signal) {
		log.Warn("Stopping by OS signal..", zap.String("signal", sig.String()))

		cancel()
	})

	defer func() {
		cancel()   // call the cancellation function after all
		oss.Stop() // stop system signals listening
	}()

	opt, optErr := redis.ParseURL(f.redisDSN)
	if optErr != nil {
		return optErr
	}

	rdb := redis.NewClient(opt).WithContext(ctx)

	defer func() { _ = rdb.Close() }()

	if pingErr := rdb.Ping(ctx).Err(); pingErr != nil {
		return pingErr
	}

	sessionTTL, parsingErr := time.ParseDuration(f.sessionTTL)
	if parsingErr != nil {
		return parsingErr
	}

	stor := storage.NewRedisStorage(ctx, rdb, sessionTTL, f.maxRequests)

	appSettings := &settings.AppSettings{
		MaxRequests:          f.maxRequests,
		SessionTTL:           sessionTTL,
		PusherKey:            f.pusher.key,         // FIXME depends on broadcast driver
		PusherCluster:        f.pusher.cluster,     // FIXME depends on broadcast driver
		IgnoreHeaderPrefixes: f.ignoreHeaderPrefix, // FIXME
	}

	var broadcaster interface {
		Publish(channel string, event broadcast.Event) error
	}

	switch f.broadcastDriver {
	case brDriverNone:
		broadcaster = &broadcast.None{}

	case brDriverPusher:
		broadcaster = broadcast.NewPusher(f.pusher.appID, f.pusher.key, f.pusher.secret, f.pusher.cluster)

	default:
		return errors.New("unsupported broadcasting driver")
	}

	// create HTTP server
	server := appHttp.NewServer(ctx, log, f.publicDir, appSettings, stor, broadcaster, rdb)

	// register server routes, middlewares, etc.
	if err := server.Register(); err != nil {
		return err
	}

	startingErrCh := make(chan error, 1) // channel for server starting error

	// start HTTP server in separate goroutine
	go func(errCh chan<- error) {
		defer close(errCh)

		fields := []zap.Field{
			zap.String("addr", f.listen.ip),
			zap.Uint16("port", f.listen.port),
			zap.String("public", f.publicDir),
			zap.Uint16("max requests", f.maxRequests),
			zap.String("session ttl", f.sessionTTL),
			zap.Strings("ignore prefixes", f.ignoreHeaderPrefix),
			zap.String("redis dsn", f.redisDSN),
			zap.String("broadcast driver", f.broadcastDriver),
		}

		log.Info("Server starting", fields...)

		if f.publicDir == "" {
			log.Warn("Path to the directory with public assets was not provided")
		}

		if err := server.Start(f.listen.ip, f.listen.port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}(startingErrCh)

	// and wait for..
	select {
	case err := <-startingErrCh: // ..server starting error
		return err

	case <-ctx.Done(): // ..or context cancellation
		log.Debug("Server stopping")

		// create context for server graceful shutdown
		ctxShutdown, ctxCancelShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer ctxCancelShutdown()

		// stop the server using created context above
		if err := server.Stop(ctxShutdown); err != nil {
			return err
		}

		// and close redis connection
		if err := rdb.Close(); err != nil {
			return err
		}
	}

	return nil
}

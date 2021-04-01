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
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	appHttp "github.com/tarampampam/webhook-tester/internal/pkg/http"
	"go.uber.org/zap"
)

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
			return run(ctx, log, f.toConfig(), f.listen.ip, f.listen.port, f.publicDir, f.redisDSN)
		},
	}

	f.init(cmd.Flags())

	return cmd
}

const serverShutdownTimeout = 5 * time.Second

// run current command.
func run( //nolint:funlen
	parentCtx context.Context,
	log *zap.Logger,
	cfg config.Config,
	ip string,
	port uint16,
	publicDir string,
	redisDSN string,
) error {
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

	var rdb *redis.Client // vam be nil, this is ok

	// establish connection with redis server, if this action is required (based on storage driver)
	if cfg.StorageDriver == config.StorageDriverRedis {
		opt, optErr := redis.ParseURL(redisDSN)
		if optErr != nil {
			return optErr
		}

		rdb = redis.NewClient(opt).WithContext(ctx)

		defer func() { _ = rdb.Close() }()

		if pingErr := rdb.Ping(ctx).Err(); pingErr != nil {
			return pingErr
		}
	}

	// create HTTP server
	server := appHttp.NewServer(log)

	// register server routes, middlewares, etc.
	if err := server.Register(ctx, cfg, publicDir, rdb); err != nil {
		return err
	}

	startingErrCh := make(chan error, 1) // channel for server starting error

	// start HTTP server in separate goroutine
	go func(errCh chan<- error) {
		defer close(errCh)

		fields := []zap.Field{
			zap.String("addr", ip),
			zap.Uint16("port", port),
			zap.String("public", publicDir),
			zap.Uint16("max requests", cfg.MaxRequests),
			zap.Duration("session ttl", cfg.SessionTTL),
			zap.Strings("ignore prefixes", cfg.IgnoreHeaderPrefixes),
			zap.String("storage driver", cfg.StorageDriver.String()),
			zap.String("broadcast driver", cfg.BroadcastDriver.String()),
		}

		if cfg.StorageDriver == config.StorageDriverRedis {
			fields = append(fields, zap.String("redis dsn", redisDSN))
		}

		log.Info("Server starting", fields...)

		if publicDir == "" {
			log.Warn("Path to the directory with public assets was not provided")
		}

		if err := server.Start(ip, port); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		if rdb != nil {
			if err := rdb.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

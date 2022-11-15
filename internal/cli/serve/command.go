// Package serve contains CLI `serve` command implementation.
package serve

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/breaker"
	"github.com/tarampampam/webhook-tester/internal/cli/shared"
	"github.com/tarampampam/webhook-tester/internal/config"
	"github.com/tarampampam/webhook-tester/internal/env"
	appHttp "github.com/tarampampam/webhook-tester/internal/http"
	"github.com/tarampampam/webhook-tester/internal/logger"
	"github.com/tarampampam/webhook-tester/internal/pubsub"
	"github.com/tarampampam/webhook-tester/internal/storage"
)

// NewCommand creates `serve` command.
func NewCommand(log *zap.Logger) *cli.Command { //nolint:funlen,gocyclo
	const (
		listenFlagName             = "listen"
		maxRequestsFlagName        = "max-requests"
		sessionTtlFlagName         = "session-ttl"
		ignoreHeaderPrefixFlagName = "ignore-header-prefix"
		maxRequestBodySizeFlagName = "max-request-body-size"
		redisDsnFlagName           = "redis-dsn"
		storageDriverFlagName      = "storage-driver"
		pubSubDriverFlagName       = "pubsub-driver"
		wsMaxClientsFlagName       = "ws-max-clients"
		wsMaxLifetimeFlagName      = "ws-max-lifetime"
		createSessionFlagName      = "create-session"
	)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s", "server"},
		Usage:   "Start HTTP server",
		Action: func(c *cli.Context) error {
			var (
				port               = c.Uint(shared.PortNumberFlag.Name)
				listen             = c.String(listenFlagName)
				maxRequests        = c.Uint(maxRequestsFlagName)
				sessionTtl         = c.Duration(sessionTtlFlagName)
				ignoreHeaderPrefix = c.StringSlice(ignoreHeaderPrefixFlagName)
				maxRequestBodySize = c.Uint(maxRequestBodySizeFlagName)
				redisDsn           = c.String(redisDsnFlagName)
				storageDriver      = c.String(storageDriverFlagName)
				pubSubDriver       = c.String(pubSubDriverFlagName)
				wsMaxClients       = c.Uint(wsMaxClientsFlagName)
				wsMaxLifetime      = c.Duration(wsMaxLifetimeFlagName)
				createSession      = c.String(createSessionFlagName)
			)

			{
				if port > math.MaxUint16 {
					return errors.New("wrong TCP port number")
				}

				if net.ParseIP(listen) == nil {
					return fmt.Errorf("wrong IP address [%s] for listening", listen)
				}

				if maxRequests > math.MaxUint16 {
					return errors.New("wrong max requests value")
				}

				_, redisDsnErr := redis.ParseURL(redisDsn)

				switch storageDriver {
				case config.StorageDriverMemory.String():
					// do nothing

				case config.StorageDriverRedis.String():
					if redisDsnErr != nil {
						return fmt.Errorf("wrong redis DSN [%s]: %w", redisDsn, redisDsnErr)
					}

				default:
					return fmt.Errorf("unsupported storage driver: %s", storageDriver)
				}

				switch pubSubDriver {
				case config.PubSubDriverMemory.String():
					// do nothing

				case config.PubSubDriverRedis.String():
					if redisDsnErr != nil {
						return fmt.Errorf("wrong redis DSN [%s]: %w", redisDsn, redisDsnErr)
					}

				default:
					return fmt.Errorf("unsupported pub/sub driver: %s", pubSubDriver)
				}

				if createSession != "" {
					if _, err := uuid.Parse(createSession); err != nil {
						return fmt.Errorf("wrong session UUID: %s", createSession)
					}
				}
			}

			var cfg = config.Config{}

			{
				cfg.MaxRequests = uint16(maxRequests)
				cfg.IgnoreHeaderPrefixes = ignoreHeaderPrefix
				cfg.MaxRequestBodySize = uint32(maxRequestBodySize)
				cfg.SessionTTL = sessionTtl

				switch storageDriver {
				case config.StorageDriverMemory.String():
					cfg.StorageDriver = config.StorageDriverMemory

				case config.StorageDriverRedis.String():
					cfg.StorageDriver = config.StorageDriverRedis
				}

				switch pubSubDriver {
				case config.PubSubDriverMemory.String():
					cfg.PubSubDriver = config.PubSubDriverMemory

				case config.PubSubDriverRedis.String():
					cfg.PubSubDriver = config.PubSubDriverRedis
				}

				cfg.WebSockets.MaxClients = uint32(wsMaxClients)
				cfg.WebSockets.MaxLifetime = wsMaxLifetime
			}

			return run(c.Context, log, cfg, listen, uint16(port), redisDsn, createSession)
		},
		Flags: []cli.Flag{
			shared.PortNumberFlag,
			&cli.StringFlag{
				Name:    listenFlagName,
				Aliases: []string{"l"},
				Usage:   "IP address to listen on",
				Value:   "0.0.0.0",
				EnvVars: []string{env.ListenAddr.String()},
			},
			&cli.UintFlag{
				Name:    maxRequestsFlagName,
				Usage:   "maximum stored requests per session (max 65535)",
				Value:   128, //nolint:gomnd
				EnvVars: []string{env.MaxSessionRequests.String()},
			},
			&cli.DurationFlag{
				Name:    sessionTtlFlagName,
				Usage:   "session lifetime (examples: 48h, 1h30m)",
				Value:   time.Hour * 168, //nolint:gomnd
				EnvVars: []string{env.SessionTTL.String()},
			},
			&cli.StringSliceFlag{
				Name:  ignoreHeaderPrefixFlagName,
				Usage: "ignore headers with the following prefixes for webhooks, case insensitive (example: 'X-Forwarded-')",
				// EnvVars: []string{}, // TODO add env var
			},
			&cli.UintFlag{
				Name:  maxRequestBodySizeFlagName,
				Usage: "maximal webhook request body size (in bytes; 0 = unlimited)",
				Value: 64 * 1024, //nolint:gomnd // 64 KiB
				// EnvVars: []string{}, // TODO add env var
			},
			&cli.StringFlag{
				// redisDSN allows to setup redis server using single string. Examples:
				//	redis://<user>:<password>@<host>:<port>/<db_number>
				//	unix://<user>:<password>@</path/to/redis.sock>?db=<db_number>
				Name:    redisDsnFlagName,
				Usage:   "redis server DSN (format: \"redis://<user>:<password>@<host>:<port>/<db_number>\")",
				Value:   "redis://127.0.0.1:6379/0",
				EnvVars: []string{env.RedisDSN.String()},
			},
			&cli.StringFlag{
				Name:    storageDriverFlagName,
				Usage:   fmt.Sprintf("storage driver (%s|%s)", config.StorageDriverMemory, config.StorageDriverRedis),
				Value:   config.StorageDriverMemory.String(),
				EnvVars: []string{env.StorageDriverName.String()},
			},
			&cli.StringFlag{
				Name:    pubSubDriverFlagName,
				Usage:   fmt.Sprintf("pub/sub driver (%s|%s)", config.PubSubDriverMemory, config.PubSubDriverRedis),
				Value:   config.PubSubDriverMemory.String(),
				EnvVars: []string{env.PubSubDriver.String()},
			},
			&cli.UintFlag{
				Name:    wsMaxClientsFlagName,
				Usage:   "maximal websocket clients count (0 = unlimited)",
				Value:   0,
				EnvVars: []string{env.WebsocketMaxClients.String()},
			},
			&cli.DurationFlag{
				Name:    wsMaxLifetimeFlagName,
				Usage:   "maximal single websocket lifetime (examples: 3h, 1h30m; 0 = unlimited)",
				Value:   time.Duration(0),
				EnvVars: []string{env.WebsocketMaxLifetime.String()},
			},
			&cli.StringFlag{
				Name:    createSessionFlagName,
				Usage:   "crete a session on server startup with this UUID (for the persistent URL, example: 00000000-0000-0000-0000-000000000000)", //nolint:lll
				EnvVars: []string{env.CreateSessionUUID.String()},
			},
		},
	}
}

const serverShutdownTimeout = 5 * time.Second

// run current command.
func run( //nolint:funlen,gocyclo
	parentCtx context.Context,
	log *zap.Logger,
	cfg config.Config,
	ip string,
	port uint16,
	redisDSN string,
	createSessionUUID string,
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

	var rdb *redis.Client // can be nil, that's ok

	// establish connection with the redis server, if this action is required (based on storage/pubsub drivers)
	if cfg.StorageDriver == config.StorageDriverRedis || cfg.PubSubDriver == config.PubSubDriverRedis {
		opt, optErr := redis.ParseURL(redisDSN)
		if optErr != nil {
			return optErr
		}

		rdb = redis.NewClient(opt).WithContext(ctx)
		redis.SetLogger(logger.NewRedisBridge(log)) // set zap logger for the redis client (globally)

		defer func() { _ = rdb.Close() }()

		if pingErr := rdb.Ping(ctx).Err(); pingErr != nil {
			return pingErr
		}
	}

	var stor storage.Storage

	// create required storage driver
	switch cfg.StorageDriver {
	case config.StorageDriverRedis:
		stor = storage.NewRedis(ctx, rdb, cfg.SessionTTL, cfg.MaxRequests)

	case config.StorageDriverMemory:
		inmemory := storage.NewInMemory(cfg.SessionTTL, cfg.MaxRequests)
		defer func() { _ = inmemory.Close() }()

		stor = inmemory

	default:
		return errors.New("unsupported storage driver") // cannot be covered by tests
	}

	if createSessionUUID != "" { // create a persistent session
		if _, err := stor.CreateSession( // persistent session defaults
			[]byte{},
			http.StatusOK,
			"text/plain; charset=utf-8",
			time.Duration(0),
			createSessionUUID,
		); err != nil {
			log.Error("cannot create persistent session", zap.Error(err))
		} else {
			log.Info("persistent session created", zap.String("uuid", createSessionUUID))
		}
	}

	var (
		pub pubsub.Publisher
		sub pubsub.Subscriber
	)

	// create required pub/sub driver
	switch cfg.PubSubDriver {
	case config.PubSubDriverRedis:
		redisPubSub := pubsub.NewRedis(ctx, rdb)
		defer func() { _ = redisPubSub.Close() }()

		pub, sub = redisPubSub, redisPubSub

	case config.PubSubDriverMemory:
		memoryPubSub := pubsub.NewInMemory()
		defer func() { _ = memoryPubSub.Close() }()

		pub, sub = memoryPubSub, memoryPubSub

	default:
		return errors.New("unsupported pub/sub driver") // cannot be covered by tests
	}

	// create HTTP server
	server := appHttp.NewServer(log)

	// register server routes, middlewares, etc.
	if err := server.Register(ctx, cfg, rdb, stor, pub, sub); err != nil {
		return err
	}

	startingErrCh := make(chan error, 1) // channel for server starting error

	// start HTTP server in separate goroutine
	go func(errCh chan<- error) {
		defer close(errCh)

		fields := []zap.Field{
			zap.String("addr", ip),
			zap.Uint16("port", port),
			zap.Uint16("max requests", cfg.MaxRequests),
			zap.Duration("session ttl", cfg.SessionTTL),
			zap.Strings("ignore prefixes", cfg.IgnoreHeaderPrefixes),
			zap.String("storage driver", cfg.StorageDriver.String()),
			zap.String("pub/sub driver", cfg.PubSubDriver.String()),
			zap.Uint32("max websocket clients", cfg.WebSockets.MaxClients),
			zap.Duration("single websocket ttl", cfg.WebSockets.MaxLifetime),
		}

		if cfg.StorageDriver == config.StorageDriverRedis {
			fields = append(fields, zap.String("redis dsn", redisDSN))
		}

		log.Info("Server starting", fields...)

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

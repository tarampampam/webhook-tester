package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast/pusher"
	apphttp "github.com/tarampampam/webhook-tester/internal/pkg/http"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage/redis"
)

const (
	// gracefully shutdown timeout.
	shutdownTimeout = time.Second * 5

	// HTTP read/write timeouts
	httpReadTimeout  = time.Second * 5
	httpWriteTimeout = time.Second * 35
)

type (
	address   string
	port      uint16
	publicDir string
)

// Command is a `serve` command.
type Command struct {
	Address            address   `required:"true" long:"listen" env:"LISTEN_ADDR" default:"0.0.0.0" description:"IP address to listen on"`                 //nolint:lll
	Port               port      `required:"true" long:"port" env:"LISTEN_PORT" default:"8080" description:"TCP port number"`                              //nolint:lll
	PublicDir          publicDir `required:"true" long:"public" env:"PUBLIC_DIR" default:"./web" description:"Directory with public assets"`               //nolint:lll
	MaxRequests        uint16    `required:"true" long:"max-requests" default:"128" env:"MAX_REQUESTS" description:"Maximum stored requests per session"`  //nolint:lll
	SessionTTLSec      uint32    `required:"true" long:"session-ttl" default:"604800" env:"SESSION_TTL" description:"Session lifetime (in seconds)"`       //nolint:lll
	IgnoreHeaderPrefix []string  `long:"ignore-header-prefix" description:"Ignore incoming webhook header prefix, case insensitive (like 'X-Forwarded-')"` //nolint:lll
	RedisHost          string    `required:"true" long:"redis-host" env:"REDIS_HOST" description:"Redis server hostname or IP address"`                    //nolint:lll
	RedisPort          port      `required:"true" long:"redis-port" default:"6379" env:"REDIS_PORT" description:"Redis server TCP port number"`            //nolint:lll
	RedisPass          string    `long:"redis-password" default:"" env:"REDIS_PASSWORD" description:"Redis server password (optional)"`                    //nolint:lll
	RedisDBNum         uint16    `required:"true" long:"redis-db-num" default:"1" env:"REDIS_DB_NUM" description:"Redis database number"`                  //nolint:lll
	RedisMaxConn       uint16    `required:"true" long:"redis-max-conn" default:"10" env:"REDIS_MAX_CONN" description:"Maximum redis connections"`         //nolint:lll
	PusherAppID        string    `long:"pusher-app-id" env:"PUSHER_APP_ID" description:"Pusher application ID"`
	PusherKey          string    `long:"pusher-key" env:"PUSHER_KEY" description:"Pusher key"`
	PusherSecret       string    `long:"pusher-secret" env:"PUSHER_SECRET" description:"Pusher secret"`
	PusherCluster      string    `long:"pusher-cluster" default:"eu" env:"PUSHER_CLUSTER" description:"Pusher cluster"`
}

// Convert struct into string representation.
func (a address) String() string   { return string(a) }
func (p port) String() string      { return strconv.FormatUint(uint64(p), 10) }
func (d publicDir) String() string { return string(d) }

// Validate address for listening on.
func (address) IsValidValue(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.New("wrong address for listening value (invalid IP address)")
	}

	return nil
}

// Validate public directory path.
func (publicDir) IsValidValue(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err
	}

	return nil
}

// Execute current command.
func (cmd *Command) Execute(_ []string) error {
	var (
		appSettings       = cmd.getAppSettings()
		ctx, cancel       = context.WithTimeout(context.Background(), shutdownTimeout)
		dataStorage       = cmd.getStorage(ctx, appSettings)
		broadcaster, bErr = cmd.getBroadcaster()
	)

	if bErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, bErr.Error())
	}

	server := apphttp.NewServer(&apphttp.ServerSettings{
		Address:                   cmd.Address.String() + ":" + cmd.Port.String(),
		WriteTimeout:              httpWriteTimeout,
		ReadTimeout:               httpReadTimeout,
		PublicAssetsDirectoryPath: cmd.PublicDir.String(),
		KeepAliveEnabled:          false,
	}, appSettings, dataStorage, broadcaster)

	server.RegisterHandlers()

	// make a channel for system signals
	signals := make(chan os.Signal, 1)

	// "Subscribe" for system signals
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// start server in a goroutine
	go func() {
		if startErr := server.Start(); startErr != http.ErrServerClosed {
			panic("Server starting error")
		}
	}()

	// listen for a signal
	<-signals

	defer func() {
		if err := dataStorage.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "storage closing error: %s\n", err)
		}

		cancel()
	}()

	if err := server.Stop(ctx); err != nil {
		return err
	}

	return nil
}

func (cmd *Command) getAppSettings() *settings.AppSettings {
	return &settings.AppSettings{
		MaxRequests:          cmd.MaxRequests,
		SessionTTL:           time.Second * time.Duration(cmd.SessionTTLSec),
		PusherKey:            cmd.PusherKey,
		PusherCluster:        cmd.PusherCluster,
		IgnoreHeaderPrefixes: cmd.IgnoreHeaderPrefix,
	}
}

func (cmd *Command) getStorage(_ context.Context, appSettings *settings.AppSettings) storage.Storage {
	return redis.NewStorage(
		cmd.RedisHost+":"+cmd.RedisPort.String(),
		cmd.RedisPass,
		int(cmd.RedisDBNum),
		int(cmd.RedisMaxConn),
		appSettings.SessionTTL,
		appSettings.MaxRequests,
	)
}

func (cmd *Command) getBroadcaster() (broadcast.Broadcaster, error) {
	if cmd.PusherAppID != "" && cmd.PusherKey != "" && cmd.PusherSecret != "" && cmd.PusherCluster != "" {
		return pusher.NewBroadcaster(cmd.PusherAppID, cmd.PusherKey, cmd.PusherSecret, cmd.PusherCluster), nil
	}

	return nil, errors.New("pusher.com cannot be registered (wrong configuration)")
}

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
	apphttp "webhook-tester/http"
	"webhook-tester/settings"
	"webhook-tester/storage/redis"
)

const (
	// gracefully shutdown timeout.
	shutdownTimeout = time.Second * 5

	// HTTP read/write timeouts
	httpReadTimeout  = time.Second * 3
	httpWriteTimeout = time.Second * 3
)

type (
	address   string
	port      uint16
	publicDir string
)

// Command is a `serve` command.
type Command struct {
	Address       address   `required:"true" long:"listen" env:"LISTEN_ADDR" default:"0.0.0.0" description:"IP address to listen on"`                //nolint:lll
	Port          port      `required:"true" long:"port" env:"LISTEN_PORT" default:"8080" description:"TCP port number"`                             //nolint:lll
	PublicDir     publicDir `required:"true" long:"public" default:"./public" description:"Directory with public assets"`                            //nolint:lll
	MaxRequests   uint16    `required:"true" long:"max-requests" default:"128" env:"MAX_REQUESTS" description:"Maximum stored requests per session"` //nolint:lll
	SessionTTLSec uint32    `required:"true" long:"session-ttl" default:"604800" env:"SESSION_TTL" description:"Session lifetime (in seconds)"`      //nolint:lll
	RedisHost     string    `required:"true" long:"redis-host" env:"REDIS_HOST" description:"Redis server hostname or IP address"`                   //nolint:lll
	RedisPort     port      `required:"true" long:"redis-port" default:"6379" env:"REDIS_PORT" description:"Redis server TCP port number"`           //nolint:lll
	RedisPass     string    `long:"redis-password" default:"" env:"REDIS_PASSWORD" description:"Optional redis server password"`                     //nolint:lll
	RedisDBNum    uint16    `required:"true" long:"redis-db-num" default:"1" env:"REDIS_DB_NUM" description:"Redis database number"`                 //nolint:lll
	RedisMaxConn  uint16    `required:"true" long:"redis-max-conn" default:"10" env:"REDIS_MAX_CONN" description:"Maximum redis connections"`        //nolint:lll
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
	appSettings := &settings.AppSettings{
		MaxRequests: cmd.MaxRequests,
		SessionTTL:  time.Second * time.Duration(cmd.SessionTTLSec),
	}

	storage := redis.NewStorage(
		cmd.RedisHost+":"+cmd.RedisPort.String(),
		cmd.RedisPass,
		int(cmd.RedisDBNum),
		int(cmd.RedisMaxConn),
		appSettings.SessionTTL,
		appSettings.MaxRequests,
	)

	server := apphttp.NewServer(&apphttp.ServerSettings{
		Address:                   cmd.Address.String() + ":" + cmd.Port.String(),
		WriteTimeout:              httpWriteTimeout,
		ReadTimeout:               httpReadTimeout,
		PublicAssetsDirectoryPath: cmd.PublicDir.String(),
		KeepAliveEnabled:          false,
	}, appSettings, storage)

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

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	// listen for a signal
	<-signals

	defer func() {
		if err := storage.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "storage closing error: %s\n", err)
		}

		cancel()
	}()

	if err := server.Stop(ctx); err != nil {
		return err
	}

	return nil
}

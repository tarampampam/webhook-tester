package serve

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	apphttp "webhook-tester/http"
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
	Address   address   `required:"true" long:"listen" env:"LISTEN_ADDR" default:"0.0.0.0" description:"IP address to listen on"` //nolint:lll
	Port      port      `required:"true" long:"port" env:"LISTEN_PORT" default:"8080" description:"TCP port number"`
	PublicDir publicDir `required:"true" long:"public" default:"./public" description:"Directory with public assets"`
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
	server := apphttp.NewServer(&apphttp.ServerSettings{
		Address:                   cmd.Address.String() + ":" + cmd.Port.String(),
		WriteTimeout:              httpWriteTimeout,
		ReadTimeout:               httpReadTimeout,
		PublicAssetsDirectoryPath: cmd.PublicDir.String(),
		KeepAliveEnabled:          false,
	})

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

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	defer func() {
		// stop any additional services right here
		cancel()
	}()

	if err := server.Stop(ctx); err != nil {
		return err
	}

	return nil
}

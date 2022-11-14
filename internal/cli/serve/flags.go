package serve

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/pflag"

	"github.com/tarampampam/webhook-tester/internal/config"
	"github.com/tarampampam/webhook-tester/internal/env"
)

type flags struct {
	listen struct {
		ip   string
		port uint16
	}

	maxRequests        uint16
	sessionTTL         time.Duration
	ignoreHeaderPrefix []string
	maxRequestBodySize uint32 // maximal webhook request body size (in bytes)

	// redisDSN allows to setup redis server using single string. Examples:
	//	redis://<user>:<password>@<host>:<port>/<db_number>
	//	unix://<user>:<password>@</path/to/redis.sock>?db=<db_number>
	redisDSN string

	storageDriver, pubSubDriver string

	websocket struct {
		maxClients  uint32
		maxLifetime time.Duration
	}
}

func (f *flags) init(flagSet *pflag.FlagSet) { //nolint:funlen
	exe, _ := os.Executable()
	exe = path.Dir(exe) //nolint:ineffassign

	flagSet.StringVarP(
		&f.listen.ip,
		"listen",
		"l",
		"0.0.0.0",
		fmt.Sprintf("IP address to listen on [$%s]", env.ListenAddr),
	)
	flagSet.Uint16VarP(
		&f.listen.port,
		"port",
		"p",
		8080, //nolint:gomnd
		fmt.Sprintf("TCP port number [$%s]", env.ListenPort),
	)
	flagSet.Uint16VarP(
		&f.maxRequests,
		"max-requests",
		"",
		128, //nolint:gomnd
		fmt.Sprintf("maximum stored requests per session (max 65535) [$%s]", env.MaxSessionRequests),
	)
	flagSet.DurationVarP(
		&f.sessionTTL,
		"session-ttl",
		"",
		time.Hour*168, //nolint:gomnd
		fmt.Sprintf("session lifetime (examples: 48h, 1h30m) [$%s]", env.SessionTTL),
	)
	flagSet.StringSliceVarP(
		&f.ignoreHeaderPrefix,
		"ignore-header-prefix",
		"",
		[]string{},
		"ignore incoming webhook header prefix, case insensitive (example: 'X-Forwarded-')",
	)
	flagSet.Uint32VarP(
		&f.maxRequestBodySize,
		"max-request-body-size",
		"",
		64*1024, //nolint:gomnd // 64 KiB
		"maximal webhook request body size (in bytes; 0 = unlimited)",
	)
	flagSet.StringVarP(
		&f.redisDSN,
		"redis-dsn",
		"",
		"redis://127.0.0.1:6379/0",
		fmt.Sprintf("redis server DSN (format: \"redis://<user>:<password>@<host>:<port>/<db_number>\") [$%s]", env.RedisDSN), //nolint:lll
	)
	flagSet.StringVarP(
		&f.storageDriver,
		"storage-driver",
		"",
		config.StorageDriverMemory.String(),
		fmt.Sprintf("storage driver (%s|%s) [$%s]", config.StorageDriverMemory, config.StorageDriverRedis, env.StorageDriverName), //nolint:lll
	)
	flagSet.StringVarP(
		&f.pubSubDriver,
		"pubsub-driver",
		"",
		config.PubSubDriverMemory.String(),
		fmt.Sprintf("pub/sub driver (%s|%s) [$%s]", config.PubSubDriverMemory, config.PubSubDriverRedis, env.PubSubDriver), //nolint:lll
	)
	flagSet.Uint32VarP(
		&f.websocket.maxClients,
		"ws-max-clients",
		"",
		0,
		fmt.Sprintf("maximal websocket clients (0 = unlimited) [$%s]", env.WebsocketMaxClients),
	)
	flagSet.DurationVarP(
		&f.websocket.maxLifetime,
		"ws-max-lifetime",
		"",
		time.Duration(0),
		fmt.Sprintf("maximal single websocket lifetime (examples: 3h, 1h30m; 0 = unlimited) [$%s]", env.WebsocketMaxLifetime),
	)
}

func (f *flags) overrideUsingEnv() error {
	if envVar, exists := env.ListenAddr.Lookup(); exists {
		f.listen.ip = envVar
	}

	if envVar, exists := env.ListenPort.Lookup(); exists {
		if p, err := strconv.ParseUint(envVar, 10, 16); err == nil {
			f.listen.port = uint16(p)
		} else {
			return fmt.Errorf("wrong TCP port environment variable [%s] value", envVar)
		}
	}

	if envVar, exists := env.MaxSessionRequests.Lookup(); exists {
		if p, err := strconv.ParseUint(envVar, 10, 16); err == nil {
			f.maxRequests = uint16(p)
		} else {
			return fmt.Errorf("wrong maximum session requests [%s] value", envVar)
		}
	}

	if envVar, exists := env.SessionTTL.Lookup(); exists {
		if d, err := time.ParseDuration(envVar); err == nil {
			f.sessionTTL = d
		} else {
			return fmt.Errorf("wrong session lifetime [%s] period", envVar)
		}
	}

	if envVar, exists := env.RedisDSN.Lookup(); exists {
		f.redisDSN = envVar
	}

	if envVar, exists := env.StorageDriverName.Lookup(); exists {
		f.storageDriver = envVar
	}

	if envVar, exists := env.PubSubDriver.Lookup(); exists {
		f.pubSubDriver = envVar
	}

	if envVar, exists := env.WebsocketMaxClients.Lookup(); exists {
		if p, err := strconv.ParseUint(envVar, 10, 32); err == nil {
			f.websocket.maxClients = uint32(p)
		} else {
			return fmt.Errorf("wrong maximal websocket clients count [%s] value", envVar)
		}
	}

	if envVar, exists := env.WebsocketMaxLifetime.Lookup(); exists {
		if d, err := time.ParseDuration(envVar); err == nil {
			f.websocket.maxLifetime = d
		} else {
			return fmt.Errorf("wrong maximal single websocket lifetime [%s] period", envVar)
		}
	}

	return nil
}

func (f *flags) validate() error {
	if net.ParseIP(f.listen.ip) == nil {
		return fmt.Errorf("wrong IP address [%s] for listening", f.listen.ip)
	}

	switch f.storageDriver {
	case config.StorageDriverMemory.String():
		// do nothing

	case config.StorageDriverRedis.String():
		if _, err := redis.ParseURL(f.redisDSN); err != nil {
			return fmt.Errorf("wrong redis DSN [%s]: %w", f.redisDSN, err)
		}

	default:
		return fmt.Errorf("unsupported storage driver: %s", f.storageDriver)
	}

	switch f.pubSubDriver {
	case config.PubSubDriverMemory.String():
		// do nothing

	case config.PubSubDriverRedis.String():
		if _, err := redis.ParseURL(f.redisDSN); err != nil {
			return fmt.Errorf("wrong redis DSN [%s]: %w", f.redisDSN, err)
		}

	default:
		return fmt.Errorf("unsupported pub/sub driver: %s", f.pubSubDriver)
	}

	return nil
}

func (f *flags) toConfig() config.Config {
	cfg := config.Config{
		MaxRequests:          f.maxRequests,
		IgnoreHeaderPrefixes: f.ignoreHeaderPrefix,
		MaxRequestBodySize:   f.maxRequestBodySize,
		SessionTTL:           f.sessionTTL,
	}

	switch f.storageDriver {
	case config.StorageDriverMemory.String():
		cfg.StorageDriver = config.StorageDriverMemory

	case config.StorageDriverRedis.String():
		cfg.StorageDriver = config.StorageDriverRedis
	}

	switch f.pubSubDriver {
	case config.PubSubDriverMemory.String():
		cfg.PubSubDriver = config.PubSubDriverMemory

	case config.PubSubDriverRedis.String():
		cfg.PubSubDriver = config.PubSubDriverRedis
	}

	cfg.WebSockets.MaxClients = f.websocket.maxClients
	cfg.WebSockets.MaxLifetime = f.websocket.maxLifetime

	return cfg
}

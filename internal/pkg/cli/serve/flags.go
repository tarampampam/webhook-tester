package serve

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/pflag"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/env"
)

type flags struct {
	listen struct {
		ip   string
		port uint16
	}

	publicDir          string // can be empty
	maxRequests        uint16
	sessionTTL         time.Duration
	ignoreHeaderPrefix []string
	maxRequestBodySize uint32 // maximal webhook request body size (in bytes)

	// redisDSN allows to setup redis server using single string. Examples:
	//	redis://<user>:<password>@<host>:<port>/<db_number>
	//	unix://<user>:<password>@</path/to/redis.sock>?db=<db_number>
	redisDSN string

	storageDriver   string
	broadcastDriver string

	pusher struct{ appID, key, secret, cluster string }
}

func (f *flags) init(flagSet *pflag.FlagSet) { //nolint:funlen
	exe, _ := os.Executable()
	exe = path.Dir(exe)

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
		8080,
		fmt.Sprintf("TCP port number [$%s]", env.ListenPort),
	)
	flagSet.StringVarP(
		&f.publicDir,
		"public",
		"",
		filepath.Join(exe, "web"),
		fmt.Sprintf("path to the directory with public assets [$%s]", env.PublicDir),
	)
	flagSet.Uint16VarP(
		&f.maxRequests,
		"max-requests",
		"",
		128,
		fmt.Sprintf("maximum stored requests per session (max 65535) [$%s]", env.MaxSessionRequests),
	)
	flagSet.DurationVarP(
		&f.sessionTTL,
		"session-ttl",
		"",
		time.Hour*168,
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
		"maximal webhook request body size (in bytes)",
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
		&f.broadcastDriver,
		"broadcast-driver",
		"",
		config.BroadcastDriverNone.String(),
		fmt.Sprintf("broadcast driver (%s|%s) [$%s]", config.BroadcastDriverNone, config.BroadcastDriverPusher, env.BroadcastDriverName), //nolint:lll
	)
	flagSet.StringVarP(
		&f.pusher.appID,
		"pusher-app-id",
		"",
		"",
		fmt.Sprintf("pusher application ID [$%s]", env.PusherAppID),
	)
	flagSet.StringVarP(
		&f.pusher.key,
		"pusher-key",
		"",
		"",
		fmt.Sprintf("pusher key [$%s]", env.PusherKey),
	)
	flagSet.StringVarP(
		&f.pusher.secret,
		"pusher-secret",
		"",
		"",
		fmt.Sprintf("pusher secret [$%s]", env.PusherSecret),
	)
	flagSet.StringVarP(
		&f.pusher.cluster,
		"pusher-cluster",
		"",
		"eu",
		fmt.Sprintf("pusher cluster [$%s]", env.PusherCluster),
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

	if envVar, exists := env.PublicDir.Lookup(); exists {
		f.publicDir = envVar
	}

	if envVar, exists := env.MaxSessionRequests.Lookup(); exists {
		if p, err := strconv.ParseUint(envVar, 10, 16); err == nil {
			f.listen.port = uint16(p)
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

	if envVar, exists := env.BroadcastDriverName.Lookup(); exists {
		f.broadcastDriver = envVar
	}

	if envVar, exists := env.PusherAppID.Lookup(); exists {
		f.pusher.appID = envVar
	}

	if envVar, exists := env.PusherKey.Lookup(); exists {
		f.pusher.key = envVar
	}

	if envVar, exists := env.PusherSecret.Lookup(); exists {
		f.pusher.secret = envVar
	}

	if envVar, exists := env.PusherCluster.Lookup(); exists {
		f.pusher.cluster = envVar
	}

	return nil
}

func (f *flags) validate() error {
	if net.ParseIP(f.listen.ip) == nil {
		return fmt.Errorf("wrong IP address [%s] for listening", f.listen.ip)
	}

	if f.publicDir != "" {
		if info, err := os.Stat(f.publicDir); err != nil || !info.Mode().IsDir() {
			return fmt.Errorf("wrong public assets directory [%s] path", f.publicDir)
		}
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

	switch f.broadcastDriver {
	case config.BroadcastDriverNone.String():
		// do nothing

	case config.BroadcastDriverPusher.String():
		if f.pusher.appID == "" {
			return errors.New("pusher application ID does not set")
		}

		if f.pusher.key == "" {
			return errors.New("pusher key does not set")
		}

		if f.pusher.secret == "" {
			return errors.New("pusher secret does not set")
		}

		if f.pusher.cluster == "" {
			return errors.New("pusher cluster does not set")
		}

	default:
		return fmt.Errorf("unsupported broadcast driver: %s", f.broadcastDriver)
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

	switch f.broadcastDriver {
	case config.BroadcastDriverNone.String():
		cfg.BroadcastDriver = config.BroadcastDriverNone

	case config.BroadcastDriverPusher.String():
		cfg.BroadcastDriver = config.BroadcastDriverPusher
	}

	cfg.Pusher.AppID = f.pusher.appID
	cfg.Pusher.Cluster = f.pusher.cluster
	cfg.Pusher.Key = f.pusher.key
	cfg.Pusher.Secret = f.pusher.secret

	return cfg
}

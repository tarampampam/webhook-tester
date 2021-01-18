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
	"github.com/tarampampam/webhook-tester/internal/pkg/env"
)

type flags struct {
	listen struct {
		ip   string
		port uint16
	}

	publicDir          string // can be empty
	maxRequests        uint16
	sessionTTL         string // duration
	ignoreHeaderPrefix []string

	// redisDSN allows to setup redis server using single string. Examples:
	//	redis://<user>:<password>@<host>:<port>/<db_number>
	//	unix://<user>:<password>@</path/to/redis.sock>?db=<db_number>
	redisDSN string

	broadcastDriver string

	pusher struct{ appID, key, secret, cluster string }
}

func (f *flags) init(flagSet *pflag.FlagSet) {
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
	flagSet.StringVarP(
		&f.sessionTTL,
		"session-ttl",
		"",
		"168h",
		fmt.Sprintf("session lifetime (examples: 48h, 1h30m) [$%s]", env.SessionTTL),
	)
	flagSet.StringSliceVarP(
		&f.ignoreHeaderPrefix,
		"ignore-header-prefix",
		"",
		[]string{},
		"ignore incoming webhook header prefix, case insensitive (can be multiple; example: 'X-Forwarded-')",
	)
	flagSet.StringVarP(
		&f.redisDSN,
		"redis-dsn",
		"",
		"redis://127.0.0.1:6379/0",
		fmt.Sprintf("redis server DSN (format: \"redis://<user>:<password>@<host>:<port>/<db_number>\") [$%s]", env.RedisDSN), //nolint:lll
	)
	flagSet.StringVarP( // TODO new (remove this comment)
		&f.broadcastDriver,
		"broadcast-driver",
		"",
		brDriverNone,
		fmt.Sprintf("broadcast driver (%s|%s) [$%s]", brDriverNone, brDriverPusher, env.BroadcastDriverName),
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
		f.sessionTTL = envVar
	}

	if envVar, exists := env.RedisDSN.Lookup(); exists {
		f.redisDSN = envVar
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

	if _, err := time.ParseDuration(f.sessionTTL); err != nil {
		return fmt.Errorf("wrong session lifetime [%s] period", f.sessionTTL)
	}

	if _, err := redis.ParseURL(f.redisDSN); err != nil {
		return fmt.Errorf("wrong redis DSN [%s]: %w", f.redisDSN, err)
	}

	switch f.broadcastDriver {
	case brDriverNone:
		// do nothing

	case brDriverPusher:
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
		return fmt.Errorf("unsupported caching engine: %s", f.broadcastDriver)
	}

	return nil
}

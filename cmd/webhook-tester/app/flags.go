package app

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v3/internal/cli"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
)

func newLogLevelFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"log-level"},
		Usage: "Logging level (" + strings.Join([]string{
			logger.DebugLevel.String(),
			logger.InfoLevel.String(),
			logger.WarnLevel.String(),
			logger.ErrorLevel.String(),
		}, "/") + ")",
		EnvVars: []string{"LOG_LEVEL"},
		Default: logger.InfoLevel.String(),
		Validator: func(_ *cli.Command, lvl string) (err error) {
			_, err = logger.ParseLevel(lvl)

			return
		},
	}
}

func newLogFormatFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"log-format"},
		Usage: "Logging format (" + strings.Join([]string{
			logger.ConsoleFormat.String(),
			logger.JSONFormat.String(),
		}, "/") + ")",
		EnvVars: []string{"LOG_FORMAT"},
		Default: logger.ConsoleFormat.String(),
		Validator: func(_ *cli.Command, fmt string) (err error) {
			_, err = logger.ParseFormat(fmt)

			return
		},
	}
}

func newHTTPAddrFlag(def string) cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"addr", "listen"},
		Usage:   "HTTP server address to listen on (IPv4 or IPv6)",
		EnvVars: []string{"HTTP_ADDR", "LISTEN_ADDR", "ADDR"},
		Default: def,
		Validator: func(_ *cli.Command, ip string) error {
			if ip == "" {
				return errors.New("missing IP address for listening")
			}

			if net.ParseIP(ip) == nil {
				return fmt.Errorf("wrong IP address [%s] for listening", ip)
			}

			return nil
		},
	}
}

func newHTTPPortFlag(def uint) cli.Flag[uint] {
	return cli.Flag[uint]{
		Names:   []string{"port"},
		Usage:   "HTTP server TCP port number",
		EnvVars: []string{"HTTP_PORT", "LISTEN_PORT", "PORT"},
		Default: def,
		Validator: func(_ *cli.Command, port uint) error {
			if port == 0 || port > 65535 {
				return fmt.Errorf("wrong TCP port number [%d]", port)
			}

			return nil
		},
	}
}

func newHTTPReadHeaderTimeoutFlag(def time.Duration) cli.Flag[time.Duration] {
	return cli.Flag[time.Duration]{
		Names: []string{"http-read-header-timeout"},
		Usage: "Maximum time allowed to read request headers; set 0 to disable " +
			"(not recommended - makes server vulnerable to Slowloris)",
		EnvVars: []string{"HTTP_READ_HEADER_TIMEOUT"},
		Default: def,
		Validator: func(_ *cli.Command, d time.Duration) error {
			if d < 0 {
				return fmt.Errorf("http read header timeout must be a non-negative duration, got %s", d)
			}

			return nil
		},
	}
}

func newHTTPReadTimeoutFlag(def time.Duration) cli.Flag[time.Duration] {
	return cli.Flag[time.Duration]{
		Names:   []string{"http-read-timeout"},
		Usage:   "Maximum duration for reading the entire request including the body; set 0 to disable",
		EnvVars: []string{"HTTP_READ_TIMEOUT"},
		Default: def,
		Validator: func(_ *cli.Command, d time.Duration) error {
			if d < 0 {
				return fmt.Errorf("http read timeout must be a non-negative duration, got %s", d)
			}

			return nil
		},
	}
}

func newHTTPIdleTimeoutFlag(def time.Duration) cli.Flag[time.Duration] {
	return cli.Flag[time.Duration]{
		Names:   []string{"http-idle-timeout"},
		Usage:   "Maximum time to wait for the next request when keep-alives are enabled; set 0 to fall back to read timeout",
		EnvVars: []string{"HTTP_IDLE_TIMEOUT"},
		Default: def,
		Validator: func(_ *cli.Command, d time.Duration) error {
			if d < 0 {
				return fmt.Errorf("http idle timeout must be a non-negative duration, got %s", d)
			}

			return nil
		},
	}
}

func newHTTPShutdownTimeoutFlag(def time.Duration) cli.Flag[time.Duration] {
	return cli.Flag[time.Duration]{
		Names:   []string{"http-shutdown-timeout"},
		Usage:   "Maximum time to wait for in-flight requests to complete on graceful shutdown; set 0 for immediate close",
		EnvVars: []string{"HTTP_SHUTDOWN_TIMEOUT"},
		Default: def,
		Validator: func(_ *cli.Command, d time.Duration) error {
			if d < 0 {
				return fmt.Errorf("http shutdown timeout must be a non-negative duration, got %s", d)
			}

			return nil
		},
	}
}

func newTLSCertFileFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"https-cert-file"},
		Usage:   "Path to the TLS certificate file for HTTPS",
		EnvVars: []string{"TLS_CERT_FILE", "HTTPS_CERT_FILE"},
		Default: "",
		Validator: func(_ *cli.Command, path string) error {
			if path == "" {
				return nil // no TLS, so no cert file is needed
			}

			if stat, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS certificate file [%s] does not exist", path)
				} else if os.IsPermission(err) {
					return fmt.Errorf("permission denied for TLS certificate file [%s]", path)
				}

				return fmt.Errorf("cannot access TLS certificate file [%s]: %w", path, err)
			} else if stat.IsDir() {
				return fmt.Errorf("TLS certificate file [%s] is a directory", path)
			}

			return nil
		},
	}
}

func newTLSKeyFileFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"https-key-file"},
		Usage:   "Path to the TLS private key file for HTTPS",
		EnvVars: []string{"TLS_KEY_FILE", "HTTPS_KEY_FILE"},
		Default: "",
		Validator: func(_ *cli.Command, path string) error {
			if path == "" {
				return nil // no TLS, so no key file is needed
			}

			if stat, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS key file [%s] does not exist", path)
				} else if os.IsPermission(err) {
					return fmt.Errorf("permission denied for TLS key file [%s]", path)
				}

				return fmt.Errorf("cannot access TLS key file [%s]: %w", path, err)
			} else if stat.IsDir() {
				return fmt.Errorf("TLS key file [%s] is a directory", path)
			}

			return nil
		},
	}
}

func newSelfSignedTLSFlag() cli.Flag[bool] {
	return cli.Flag[bool]{
		Names:   []string{"self-signed-tls"},
		Usage:   "Generate and use a self-signed TLS certificate for HTTPS (ignored if TLS cert/key files are provided)",
		EnvVars: []string{"SELF_SIGNED_CERT"},
		Default: false,
	}
}

func newRedisServerDsnFlag(def string) cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"redis-dsn"},
		Usage: "Redis-like (redis/reydb) server DSN (e.g. redis://user:pass@host:port/db or " +
			"unix://user:pass@/var/run/redis/redis.sock?db=0)",
		EnvVars: []string{"REDIS_DSN"},
		Default: def,
		Validator: func(_ *cli.Command, dsn string) error {
			if dsn == "" {
				return nil // no Redis, so no DSN is needed
			}

			if _, err := redis.ParseURL(dsn); err != nil {
				return fmt.Errorf("invalid Redis DSN [%s]: %w", dsn, err)
			}

			return nil
		},
	}
}

func newPubSubDriverFlag(def pubSubDriverName) cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"pubsub-driver"},
		Usage: "Pub/Sub driver to use for broadcasting events via WebSocket (supported: " + strings.Join([]string{
			string(pubSubDriverMemory),
			string(pubSubDriverRedis),
		}, "/") + ")",
		EnvVars: []string{"PUBSUB_DRIVER"},
		Default: string(def),
		Validator: func(_ *cli.Command, name string) error {
			switch pubSubDriverName(name) {
			case pubSubDriverMemory, pubSubDriverRedis:
				return nil
			default:
				return fmt.Errorf("unknown Pub/Sub driver name: %q", name)
			}
		},
	}
}

func newStorageDriverFlag(def storageDriverName) cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"storage-driver"},
		Usage: "Storage driver to use for storing received requests (supported: " + strings.Join([]string{
			string(storageDriverMemory),
			string(storageDriverRedis),
			string(storageDriverFS),
		}, "/") + ")",
		EnvVars: []string{"STORAGE_DRIVER"},
		Default: string(def),
		Validator: func(_ *cli.Command, name string) error {
			switch storageDriverName(name) {
			case storageDriverMemory, storageDriverRedis, storageDriverFS:
				return nil
			default:
				return fmt.Errorf("unknown storage driver name: %q", name)
			}
		},
	}
}

func newSessionTTLFlag(def time.Duration) cli.Flag[time.Duration] {
	return cli.Flag[time.Duration]{
		Names:   []string{"session-ttl"},
		Usage:   "Time-to-live for sessions (e.g. 500ms, 145s, 1h30m; set 0 for no expiration)",
		EnvVars: []string{"SESSION_TTL"},
		Default: def,
		Validator: func(_ *cli.Command, ttl time.Duration) error {
			if ttl < 0 {
				return fmt.Errorf("session TTL must be a positive duration, got %s", ttl)
			}

			return nil
		},
	}
}

func newMaxRequestBodySizeFlag(def uint) cli.Flag[uint] {
	return cli.Flag[uint]{
		Names:   []string{"max-request-body-size"},
		Usage:   "Maximum size of the request body in bytes (set 0 for unlimited)",
		EnvVars: []string{"MAX_REQUEST_BODY_SIZE"},
		Default: def,
		Validator: func(_ *cli.Command, size uint) error {
			const maxValue uint = math.MaxUint16

			if size > maxValue {
				return fmt.Errorf("max request body size [%d] is too large; maximum allowed is %d", size, maxValue)
			}

			return nil
		},
	}
}

func newAutoCreateSessionsFlag() cli.Flag[bool] {
	return cli.Flag[bool]{
		Names:   []string{"auto-create-sessions"},
		Usage:   "Automatically create new sessions for incoming requests",
		EnvVars: []string{"AUTO_CREATE_SESSION"},
	}
}

func newFSStorageDirFlag(def string) cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"fs-storage-dir"},
		Usage:   "Directory path for storing received requests when using filesystem storage driver",
		EnvVars: []string{"FS_STORAGE_DIR"},
		Default: def,
		Validator: func(_ *cli.Command, path string) error {
			if path == "" {
				return nil
			}

			if stat, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("filesystem storage directory [%s] does not exist", path)
				} else if os.IsPermission(err) {
					return fmt.Errorf("permission denied for filesystem storage directory [%s]", path)
				}

				return fmt.Errorf("cannot access filesystem storage directory [%s]: %w", path, err)
			} else if !stat.IsDir() {
				return fmt.Errorf("filesystem storage path [%s] is not a directory", path)
			}

			return nil
		},
	}
}

func newMaxRequestsFlag(def uint) cli.Flag[uint] {
	return cli.Flag[uint]{
		Names:   []string{"max-requests"},
		Usage:   "Maximum number of requests to store in memory (set 0 for unlimited)",
		EnvVars: []string{"MAX_REQUESTS"},
		Default: def,
		Validator: func(_ *cli.Command, v uint) error {
			const maxValue uint = math.MaxUint16

			if v > maxValue {
				return fmt.Errorf("max requests value [%d] is too large; maximum allowed is %d", v, maxValue)
			}

			return nil
		},
	}
}

func newPublicURLRootFlag(def string) cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"public-url-root"},
		Usage: "Public URL root override for webhook URLs (e.g., http://webhook-tester.k8s.internal); " +
			"if not set, the URL shown in the UI is based on the browser's location",
		EnvVars: []string{"PUBLIC_URL_ROOT"},
		Default: def,
		Validator: func(_ *cli.Command, v string) error {
			if v == "" {
				return nil // no public URL root, so no validation is needed
			}

			u, err := url.Parse(v)
			if err != nil {
				return fmt.Errorf("invalid public URL root [%s]: %w", v, err)
			} else if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("invalid public URL root [%s]: scheme must be http or https", v)
			} else if u.Host == "" {
				return fmt.Errorf("invalid public URL root [%s]: missing host", v)
			}

			return nil
		},
	}
}

func newTunnelDriverFlag(def tunnelDriverName) cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"tunnel-driver"},
		Usage: "Tunnel driver to use for exposing the server to the public internet (supported: " + strings.Join([]string{
			string(tunnelDriverNgrok),
		}, "/") + ")",
		EnvVars: []string{"TUNNEL_DRIVER"},
		Default: string(def),
		Validator: func(_ *cli.Command, name string) error {
			if name == "" {
				return nil // no tunnel driver, so no validation is needed
			}

			switch tunnelDriverName(name) {
			case tunnelDriverNgrok:
				return nil
			default:
				return fmt.Errorf("unknown tunnel driver name: %q", name)
			}
		},
	}
}

func newTunnelURLFlag(def string) cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"tunnel-url"},
		Usage:   "Public URL to use for the tunnel (for ngrok, register it first at https://dashboard.ngrok.com/domains)",
		EnvVars: []string{"TUNNEL_URL"},
		Default: def,
		Validator: func(_ *cli.Command, v string) error {
			if v == "" {
				return nil // no tunnel URL, so no validation is needed
			}

			u, err := url.Parse(v)
			if err != nil {
				return fmt.Errorf("invalid tunnel URL [%s]: %w", v, err)
			} else if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("invalid tunnel URL [%s]: scheme must be http or https", v)
			} else if u.Host == "" {
				return fmt.Errorf("invalid tunnel URL [%s]: missing host", v)
			}

			return nil
		},
	}
}

func newTunnelNgrokAuthTokenFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names: []string{"ngrok-auth-token"},
		Usage: "Ngrok auth token for tunnel authentication (create a new one at " +
			"https://dashboard.ngrok.com/authtokens/new)",
		EnvVars: []string{"NGROK_AUTHTOKEN"},
		Default: "",
	}
}

func newUseLiveEndpointFlag() cli.Flag[bool] {
	return cli.Flag[bool]{
		Names: []string{"use-live-frontend"},
		Usage: "Serve the frontend assets from the filesystem instead of embedded ones (useful for development)",
	}
}

func newSessionsFileFlag() cli.Flag[string] {
	return cli.Flag[string]{
		Names:   []string{"sessions-file"},
		Usage:   "Path to a JSON file with pre-configured sessions to create on startup (existing sessions are skipped)",
		EnvVars: []string{"SESSIONS_FILE"},
		Default: "",
		Validator: func(_ *cli.Command, path string) error {
			if path == "" {
				return nil
			}

			if stat, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("sessions file [%s] does not exist", path)
				} else if os.IsPermission(err) {
					return fmt.Errorf("permission denied for sessions file [%s]", path)
				}

				return fmt.Errorf("cannot access sessions file [%s]: %w", path, err)
			} else if stat.IsDir() {
				return fmt.Errorf("sessions file [%s] is a directory", path)
			}

			return nil
		},
	}
}

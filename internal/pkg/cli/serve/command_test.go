package serve_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kami-zh/go-capturer"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/cli/serve"
	"go.uber.org/zap"
)

func TestProperties(t *testing.T) {
	cmd := serve.NewCommand(context.Background(), zap.NewNop())

	assert.Equal(t, "serve", cmd.Use)
	assert.ElementsMatch(t, []string{"s", "server"}, cmd.Aliases)
	assert.NotNil(t, cmd.RunE)
}

func TestFlags(t *testing.T) {
	cmd := serve.NewCommand(context.Background(), zap.NewNop())
	exe, _ := os.Executable()
	exe = path.Dir(exe)

	cases := []struct {
		giveName      string
		wantShorthand string
		wantDefault   string
	}{
		{giveName: "listen", wantShorthand: "l", wantDefault: "0.0.0.0"},
		{giveName: "port", wantShorthand: "p", wantDefault: "8080"},
		{giveName: "public", wantShorthand: "", wantDefault: filepath.Join(exe, "web")},
		{giveName: "max-requests", wantShorthand: "", wantDefault: "128"},
		{giveName: "session-ttl", wantShorthand: "", wantDefault: "168h0m0s"},
		{giveName: "ignore-header-prefix", wantShorthand: "", wantDefault: "[]"},
		{giveName: "max-request-body-size", wantShorthand: "", wantDefault: "65536"},
		{giveName: "redis-dsn", wantShorthand: "", wantDefault: "redis://127.0.0.1:6379/0"},
		{giveName: "storage-driver", wantShorthand: "", wantDefault: "memory"},
		{giveName: "pubsub-driver", wantShorthand: "", wantDefault: "memory"},
		{giveName: "ws-max-clients", wantShorthand: "", wantDefault: "0"},
		{giveName: "ws-max-lifetime", wantShorthand: "", wantDefault: "0s"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.giveName, func(t *testing.T) {
			flag := cmd.Flag(tt.giveName)

			if flag == nil {
				assert.Failf(t, "flag not found", "flag [%s] was not found", tt.giveName)

				return
			}

			assert.Equal(t, tt.wantShorthand, flag.Shorthand)
			assert.Equal(t, tt.wantDefault, flag.DefValue)
		})
	}
}

func TestSuccessfulFlagsPreparing(t *testing.T) {
	cmd := serve.NewCommand(context.Background(), zap.NewNop())
	cmd.SetArgs([]string{"--public", ""})

	var executed bool

	cmd.RunE = func(*cobra.Command, []string) error {
		executed = true

		return nil
	}

	output := capturer.CaptureOutput(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Empty(t, output)
	assert.True(t, executed)
}

func TestFlagsWorkingWithoutCommandExecution(t *testing.T) {
	for _, tt := range []struct {
		name             string
		giveEnv          map[string]string
		giveArgs         []string
		wantErrorStrings []string
	}{
		{
			name: "Listen Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"-l", "256.256.256.256", // 255 is max
			},
			wantErrorStrings: []string{"wrong IP address", "256.256.256.256"},
		},
		{
			name:    "Listen Flag Wrong Env Value",
			giveEnv: map[string]string{"LISTEN_ADDR": "256.256.256.256"}, // 255 is max
			giveArgs: []string{
				"--public", "",
				"-l", "0.0.0.0", // `-l` flag must be ignored
			},
			wantErrorStrings: []string{"wrong IP address", "256.256.256.256"},
		},
		{
			name: "Port Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"-p", "65536", // 65535 is max
			},
			wantErrorStrings: []string{"invalid argument", "65536", "value out of range"},
		},
		{
			name:    "Port Flag Wrong Env Value",
			giveEnv: map[string]string{"LISTEN_PORT": "65536"}, // 65535 is max
			giveArgs: []string{
				"--public", "",
				"-p", "8090", // `-p` flag must be ignored
			},
			wantErrorStrings: []string{"wrong TCP port", "environment variable", "65536"},
		},
		{
			name: "Public Dir Flag Wrong Argument",
			giveArgs: []string{
				"--public", "/tmp/nonexistent/bar/baz",
			},
			wantErrorStrings: []string{"wrong public assets directory", "/tmp/nonexistent/bar/baz"},
		},
		{
			name:    "Public Dir Flag Wrong Env Value",
			giveEnv: map[string]string{"PUBLIC_DIR": "/tmp/nonexistent/bar/baz"},
			giveArgs: []string{
				"--public", ".", // `--public` flag must be ignored
			},
			wantErrorStrings: []string{"wrong public assets directory", "/tmp/nonexistent/bar/baz"},
		},
		{
			name: "Storage Driver Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--storage-driver", "foobar",
			},
			wantErrorStrings: []string{"unsupported storage driver", "foobar"},
		},
		{
			name:    "Storage Driver Flag Wrong Env Value",
			giveEnv: map[string]string{"STORAGE_DRIVER": "barbaz"},
			giveArgs: []string{
				"--public", "",
				"--storage-driver", "memory",
			},
			wantErrorStrings: []string{"unsupported storage driver", "barbaz"},
		},
		{
			name: "PubSub Driver Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--pubsub-driver", "foobar",
			},
			wantErrorStrings: []string{"unsupported pub/sub driver", "foobar"},
		},
		{
			name:    "PubSub Driver Flag Wrong Env Value",
			giveEnv: map[string]string{"PUBSUB_DRIVER": "barbaz"},
			giveArgs: []string{
				"--public", "",
				"--pubsub-driver", "foobar",
			},
			wantErrorStrings: []string{"unsupported pub/sub driver", "barbaz"},
		},
		{
			name: "PubSub Redis Driver Flag With Wrong Redis DSN",
			giveArgs: []string{
				"--public", "",
				"--pubsub-driver", "redis",
				"--redis-dsn", "foo://bar",
			},
			wantErrorStrings: []string{"wrong redis DSN", "foo://bar"},
		},
		{
			name: "Redis DSN Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--storage-driver", "redis",
				"--redis-dsn", "foo://bar",
			},
			wantErrorStrings: []string{"wrong redis DSN", "foo://bar"},
		},
		{
			name:    "Redis DSN Flag Wrong Env Value",
			giveEnv: map[string]string{"REDIS_DSN": "bar://baz"},
			giveArgs: []string{
				"--public", "",
				"--storage-driver", "redis",
				"--redis-dsn", "foo://123.123.123.123:1234/0", // `--redis-dsn` flag must be ignored
			},
			wantErrorStrings: []string{"wrong redis DSN", "bar://baz"},
		},
		{
			name: "Max Requests Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--max-requests", "65536", // 65535 max
			},
			wantErrorStrings: []string{"invalid argument", "65536", "value out of range"},
		},
		{
			name:    "Max Requests Flag Wrong Env Value",
			giveEnv: map[string]string{"MAX_REQUESTS": "65536"},
			giveArgs: []string{
				"--public", "",
				"--max-requests", "128", // `--max-requests` flag must be ignored
			},
			wantErrorStrings: []string{"wrong maximum session requests", "65536"},
		},
		{
			name: "Session TTL Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--session-ttl", "1d", // wrong
			},
			wantErrorStrings: []string{"invalid argument", "1d"},
		},
		{
			name:    "Session TTL Flag Wrong Env Value",
			giveEnv: map[string]string{"SESSION_TTL": "2d"},
			giveArgs: []string{
				"--public", "",
				"--session-ttl", "1h",
			},
			wantErrorStrings: []string{"wrong session lifetime", "2d"},
		},
		{
			name: "Maximal Websocket Clients Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--ws-max-clients", "-111",
			},
			wantErrorStrings: []string{"invalid argument", "-111", "ws-max-clients"},
		},
		{
			name:    "Maximal Websocket Clients Flag Wrong Env Value",
			giveEnv: map[string]string{"WS_MAX_CLIENTS": "-111"},
			giveArgs: []string{
				"--public", "",
				"--ws-max-clients", "123", // correct value
			},
			wantErrorStrings: []string{"wrong maximal websocket clients count", "-111"},
		},
		{
			name: "Maximal Websocket TTL Flag Wrong Argument",
			giveArgs: []string{
				"--public", "",
				"--ws-max-lifetime", "2d",
			},
			wantErrorStrings: []string{"invalid argument", "2d", "ws-max-lifetime"},
		},
		{
			name:    "Maximal Websocket TTL Flag Wrong Env Value",
			giveEnv: map[string]string{"WS_MAX_LIFETIME": "2d"},
			giveArgs: []string{
				"--public", "",
				"--ws-max-lifetime", "1h", // correct value
			},
			wantErrorStrings: []string{"wrong maximal single websocket lifetime", "2d"},
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmd := serve.NewCommand(context.Background(), zap.NewNop())
			cmd.SetArgs(tt.giveArgs)

			var executed bool

			cmd.RunE = func(*cobra.Command, []string) error {
				executed = true

				return nil
			}

			for k, v := range tt.giveEnv {
				assert.NoError(t, os.Setenv(k, v))
			}

			output := capturer.CaptureStderr(func() {
				assert.Error(t, cmd.Execute())
			})

			for k := range tt.giveEnv {
				assert.NoError(t, os.Unsetenv(k))
			}

			assert.False(t, executed)

			for _, want := range tt.wantErrorStrings {
				assert.Contains(t, output, want)
			}
		})
	}
}

func getRandomTCPPort(t *testing.T) (int, error) {
	t.Helper()

	// zero port means randomly (os) chosen port
	l, err := net.Listen("tcp", ":0") //nolint:gosec
	if err != nil {
		return 0, err
	}

	port := l.Addr().(*net.TCPAddr).Port

	if closingErr := l.Close(); closingErr != nil {
		return 0, closingErr
	}

	return port, nil
}

func checkTCPPortIsBusy(t *testing.T, port int) bool {
	t.Helper()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return true
	}

	_ = l.Close()

	return false
}

func startAndStopServer(t *testing.T, port int, args []string) string {
	var (
		output     string
		executedCh = make(chan struct{})
	)

	// start HTTP server
	go func(ch chan<- struct{}) {
		defer close(ch)

		output = capturer.CaptureStderr(func() {
			// create command with valid flags to run
			log, _ := zap.NewDevelopment()
			cmd := serve.NewCommand(context.Background(), log)
			cmd.SilenceUsage = true
			cmd.SetArgs(args)

			assert.NoError(t, cmd.Execute())
		})

		ch <- struct{}{}
	}(executedCh)

	portBusyCh := make(chan struct{})

	// check port "busy" (by HTTP server) state
	go func(ch chan<- struct{}) {
		defer close(ch)

		for i := 0; i < 2000; i++ {
			if checkTCPPortIsBusy(t, port) {
				ch <- struct{}{}

				return
			}

			<-time.After(time.Millisecond * 2)
		}

		t.Error("port opening timeout exceeded")
	}(portBusyCh)

	<-portBusyCh // wait for server starting

	// send OS signal for server stopping
	proc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	assert.NoError(t, proc.Signal(syscall.SIGINT)) // send the signal

	<-executedCh // wait until server has been stopped

	return output
}

func TestSuccessfulCommandRunningUsingRedisDrivers(t *testing.T) {
	// get TCP port number for a test
	port, err := getRandomTCPPort(t)
	assert.NoError(t, err)

	// start mini-redis
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	output := startAndStopServer(t, port, []string{
		"--public", "",
		"--port", strconv.Itoa(port),
		"--storage-driver", "redis",
		"--pubsub-driver", "redis",
		"--redis-dsn", fmt.Sprintf("redis://127.0.0.1:%s/0", mini.Port()),
	})

	assert.Contains(t, output, "Server starting")
	assert.Contains(t, output, "Stopping by OS signal")
	assert.Contains(t, output, "Server stopping")
}

func TestSuccessfulCommandRunningUsingMemoryDrivers(t *testing.T) {
	// get TCP port number for a test
	port, err := getRandomTCPPort(t)
	assert.NoError(t, err)

	// start mini-redis
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	output := startAndStopServer(t, port, []string{
		"--public", "",
		"--port", strconv.Itoa(port),
		"--storage-driver", "memory",
		"--pubsub-driver", "memory",
		"--redis-dsn", fmt.Sprintf("redis://127.0.0.1:%s/0", mini.Port()),
	})

	assert.Contains(t, output, "Server starting")
	assert.Contains(t, output, "Stopping by OS signal")
	assert.Contains(t, output, "Server stopping")
}

func TestRunningUsingBusyPortFailing(t *testing.T) {
	port, err := getRandomTCPPort(t)
	assert.NoError(t, err)

	// start mini-redis
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	// occupy a TCP port
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	assert.NoError(t, err)

	defer func() { assert.NoError(t, l.Close()) }()

	// create command with valid flags to run
	cmd := serve.NewCommand(context.Background(), zap.NewNop())
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{
		"--public", "",
		"--port", strconv.Itoa(port),
		"--storage-driver", "redis",
		"--pubsub-driver", "redis",
		"--redis-dsn", fmt.Sprintf("redis://127.0.0.1:%s/0", mini.Port()),
	})

	executedCh := make(chan struct{})

	// start HTTP server
	go func(ch chan<- struct{}) {
		defer close(ch)

		err := cmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address already in use")

		ch <- struct{}{}
	}(executedCh)

	<-executedCh // wait until server has been stopped
}

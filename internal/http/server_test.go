package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appHttp "github.com/tarampampam/webhook-tester/internal/http"
)

func getRandomTCPPort(t *testing.T) uint16 {
	t.Helper()

	l, err := net.Listen("tcp", ":0") //nolint:gosec // zero port means randomly (os) chosen port
	if err != nil {
		panic(err)
	}

	_ = l.Close()
	runtime.Gosched()

	return uint16(l.Addr().(*net.TCPAddr).Port)
}

func checkTCPPortIsBusy(t *testing.T, port uint16) bool {
	t.Helper()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		return true
	}

	_ = l.Close()
	runtime.Gosched()

	return false
}

func TestServer_StartAndStop(t *testing.T) {
	var (
		s    = appHttp.NewServer(zap.NewNop())
		port = getRandomTCPPort(t)
	)

	assert.False(t, checkTCPPortIsBusy(t, port))

	go func() {
		startingErr := s.Start("", port)

		if !errors.Is(startingErr, http.ErrServerClosed) {
			require.NoError(t, startingErr)
		}
	}()

	var tick = time.NewTicker(time.Microsecond * 10)
	defer tick.Stop()

	for i := 0; ; i++ {
		if i > 100 {
			t.Fatal("too many attempts for server start checking")
		}

		<-tick.C

		if checkTCPPortIsBusy(t, port) {
			break
		}
	}

	assert.True(t, checkTCPPortIsBusy(t, port))
	assert.NoError(t, s.Stop(context.Background()))

	for i := 0; ; i++ {
		if i > 100 {
			t.Fatal("too many attempts for server stop checking")
		}

		<-tick.C

		if !checkTCPPortIsBusy(t, port) {
			break
		}
	}
}

package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type LocalTunnel struct {
	httpClient httpClient
	log        *zap.Logger
}

type LocalTunnelOption func(*LocalTunnel)

func WithLocalTunnelHTTPClient(client httpClient) LocalTunnelOption {
	return func(lt *LocalTunnel) { lt.httpClient = client }
}

func WithLocalTunnelLogger(log *zap.Logger) LocalTunnelOption {
	return func(lt *LocalTunnel) { lt.log = log }
}

func NewLocalTunnel(opts ...LocalTunnelOption) *LocalTunnel {
	var lt = LocalTunnel{
		httpClient: &http.Client{Timeout: 30 * time.Second}, //nolint:mnd
		log:        zap.NewNop(),
	}

	for _, opt := range opts {
		opt(&lt)
	}

	return &lt
}

func (lt *LocalTunnel) Start(pCtx context.Context, localPort uint16) (*url.URL, func(), error) {
	var (
		noop        = func() { /* do nothing */ }
		ctx, cancel = context.WithCancel(pCtx)
	)

	// register a new tunnel
	var remoteAddr, publicUrl, maxConn, regErr = lt.register(ctx)
	if regErr != nil {
		cancel()

		return nil, noop, fmt.Errorf("failed to register a tunnel: %w", regErr)
	}

	remoteConnPool, closeRemotePool := NewConnectionsPool(
		ctx,
		remoteAddr,
		maxConn,
		WithConnectionsPoolErrorsHandler(func(err error) {
			lt.log.Warn("Failed to establish a connection to the LocalTunnel server", zap.Error(err))
		}),
	)

	localConnPool, closeLocalPool := NewConnectionsPool(
		ctx,
		fmt.Sprintf("127.0.0.1:%d", localPort),
		maxConn,
		WithConnectionsPoolErrorsHandler(func(err error) {
			lt.log.Error("Failed to establish a connection to the local server", zap.Error(err))
		}),
	)

	for range maxConn {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := lt.proxy(ctx, &localConnPool, &remoteConnPool); err != nil {
						lt.log.Error("Failed to proxy the connection", zap.Error(err))
					}
				}
			}
		}()
	}

	// on success, return a function to stop the tunnel
	return publicUrl, sync.OnceFunc(func() {
		cancel()          // cancel the context
		closeRemotePool() // close the remote connection pool
		closeLocalPool()  // close the local connection pool
	}), nil
}

func (lt *LocalTunnel) register(ctx context.Context) ( /* remoteAddr */ string /* publicUrl */, *url.URL /* maxConn */, uint, error) {
	var req, reqErr = http.NewRequestWithContext(ctx, http.MethodGet, "https://localtunnel.me/?new", nil)
	if reqErr != nil {
		return "", nil, 0, fmt.Errorf("failed to create a request: %w", reqErr)
	}

	// set headers to bypass the reminder (HTML) page
	req.Header.Set("Bypass-Tunnel-Reminder", "true")
	req.Header.Set("User-Agent", "Go-http-client") // do NOT use here something that looks like a browser

	// send the request to the localtunnel server
	var resp, respErr = lt.httpClient.Do(req)
	if respErr != nil {
		return "", nil, 0, fmt.Errorf("failed to request a new tunnel: %w", respErr)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var reply struct {
		ID           string `json:"id"`
		Port         int    `json:"port"`
		MaxConnCount int    `json:"max_conn_count"`
		URL          string `json:"url"`
	}

	// example: {"id":"eighty-deer-exist","port":30415,"max_conn_count":10,"url":"https://eighty-deer-exist.loca.lt"}
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return "", nil, 0, fmt.Errorf("failed to decode a response: %w", err)
	}

	// parse the public URL
	u, uErr := url.Parse(reply.URL)
	if uErr != nil || u.Host == "" || u.Scheme == "" {
		return "", nil, 0, fmt.Errorf("failed to parse a public URL (%s): %w", reply.URL, uErr)
	}

	var maxConn uint

	// in case the server returned an invalid value, use the default one (1)
	if reply.MaxConnCount <= 0 {
		maxConn = 1
	} else {
		maxConn = uint(reply.MaxConnCount)
	}

	return fmt.Sprintf("localtunnel.me:%d", reply.Port), u, maxConn, nil
}

func (lt *LocalTunnel) proxy(ctx context.Context, localPool, remotePool *ConnectionsPool) error {
	local, localOk := localPool.Get(ctx)
	if !localOk {
		return fmt.Errorf("failed to get a local connection")
	}

	defer local.Release()

	remote, remoteOk := remotePool.Get(ctx)
	if !remoteOk {
		return fmt.Errorf("failed to get a remote connection")
	}

	defer remote.Release()

	var eg, _ = errgroup.WithContext(ctx)

	eg.Go(func() (err error) { _, err = io.Copy(remote.Conn, local.Conn); return })
	eg.Go(func() (err error) { _, err = io.Copy(local.Conn, remote.Conn); return })

	return eg.Wait()
}

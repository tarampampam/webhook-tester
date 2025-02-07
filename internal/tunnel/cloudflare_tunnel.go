package tunnel

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/cloudflare/cloudflared/cmd/cloudflared/cliutil"
	"github.com/cloudflare/cloudflared/config"
	"github.com/cloudflare/cloudflared/connection"
	"github.com/cloudflare/cloudflared/edgediscovery"
	"github.com/cloudflare/cloudflared/edgediscovery/allregions"
	"github.com/cloudflare/cloudflared/features"
	"github.com/cloudflare/cloudflared/ingress"
	"github.com/cloudflare/cloudflared/metrics"
	"github.com/cloudflare/cloudflared/orchestration"
	"github.com/cloudflare/cloudflared/signal"
	"github.com/cloudflare/cloudflared/supervisor"
	"github.com/cloudflare/cloudflared/tlsconfig"
	"github.com/cloudflare/cloudflared/tunnelrpc/pogs"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

const (
	// Default configuration values
	defaultHTTPTimeout          = 15 * time.Second
	defaultGracePeriod          = 30
	defaultHAConnections        = 4
	defaultRetries              = 5
	defaultMaxEdgeAddrRetries   = 8
	defaultRPCTimeout           = 5 * time.Second
	defaultReconnectChannelSize = 4
	defaultQUICConnFlowLimit    = 30 * (1 << 20) // 30MB
	defaultQUICStreamFlowLimit  = 6 * (1 << 20)  // 6MB

	// API endpoints
	quickTunnelAPIEndpoint = "https://api.trycloudflare.com"
)

// Build information, set during compilation
var (
	Version   = "DEV"
	BuildTime = "unknown"
	BuildType = ""
	bInfo     = cliutil.GetBuildInfo(BuildType, Version)
)

// Shutdown channel for graceful termination
var graceShutdownC chan struct{}

// TunnelError represents a custom error type for tunnel operations
type TunnelError struct {
	Op  string
	Err error
}

func (e *TunnelError) Error() string {
	return fmt.Sprintf("tunnel operation %s failed: %v", e.Op, e.Err)
}

// Config holds the configuration for starting a Cloudflare tunnel
type Config struct {
	Bind    string
	Tunnel  *connection.TunnelProperties
	Context context.Context
}

func createLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

// startCloudflareTunnel initializes and starts a Cloudflare tunnel
func startCloudflareTunnel(bind string, tunnel *connection.TunnelProperties) (shutdown func(), err error) {
	cfg := &Config{
		Bind:    bind,
		Tunnel:  tunnel,
		Context: context.Background(),
	}

	metrics.RegisterBuildInfo(BuildType, BuildTime, Version)

	logTransport := createLogger()

	observer := connection.NewObserver(logTransport, logTransport)

	orchestrator, err := createOrchestrator(cfg, logTransport)
	if err != nil {
		return nil, &TunnelError{Op: "create_orchestrator", Err: err}
	}

	tunnelConfig, err := createTunnelConfig(cfg, logTransport, observer)
	if err != nil {
		return nil, &TunnelError{Op: "create_tunnel_config", Err: err}
	}

	connectedSignal := signal.New(make(chan struct{}))
	reconnectCh := make(chan supervisor.ReconnectSignal, defaultReconnectChannelSize)

	go supervisor.StartTunnelDaemon(
		cfg.Context,
		tunnelConfig,
		orchestrator,
		connectedSignal,
		reconnectCh,
		graceShutdownC,
	)

	return func() {
		graceShutdownC <- struct{}{}
	}, nil
}

// createOrchestrator initializes the tunnel orchestrator
func createOrchestrator(cfg *Config, log *zerolog.Logger) (*orchestration.Orchestrator, error) {
	ing, err := ingress.ParseIngress(&config.Configuration{
		Ingress: []config.UnvalidatedIngressRule{{Service: cfg.Bind}},
	})
	if err != nil {
		return nil, errors.Wrap(err, "parse ingress configuration")
	}

	return orchestration.NewOrchestrator(
		cfg.Context,
		&orchestration.Config{
			Ingress:            &ing,
			WarpRouting:        ingress.NewWarpRoutingConfig(&config.WarpRoutingConfig{}),
			ConfigurationFlags: map[string]string{},
			WriteTimeout:       0,
		},
		[]pogs.Tag{},
		[]ingress.Rule{},
		log,
	)
}

// createTunnelConfig builds the tunnel configuration
func createTunnelConfig(cfg *Config, log *zerolog.Logger, observer *connection.Observer) (*supervisor.TunnelConfig, error) {
	clientID, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "generate connector UUID")
	}

	protocolSelector, edgeTLSConfigs, err := setupProtocolConfig(log)
	if err != nil {
		return nil, err
	}

	cfg.Tunnel.Client = pogs.ClientInfo{
		ClientID: clientID[:],
		Features: []string{},
		Version:  bInfo.Version(),
		Arch:     bInfo.OSArch(),
	}

	return &supervisor.TunnelConfig{
		GracePeriod:                         defaultGracePeriod,
		ReplaceExisting:                     false,
		OSArch:                              runtime.GOOS + "_" + runtime.GOARCH,
		ClientID:                            clientID.String(),
		EdgeAddrs:                           []string{},
		Region:                              "",
		EdgeIPVersion:                       allregions.Auto,
		EdgeBindAddr:                        nil,
		HAConnections:                       defaultHAConnections,
		IsAutoupdated:                       false,
		LBPool:                              "",
		Tags:                                []pogs.Tag{},
		Log:                                 log,
		LogTransport:                        log,
		Observer:                            observer,
		ReportedVersion:                     "embedded-go-test",
		Retries:                             defaultRetries,
		RunFromTerminal:                     true,
		NamedTunnel:                         cfg.Tunnel,
		ProtocolSelector:                    protocolSelector,
		EdgeTLSConfigs:                      edgeTLSConfigs,
		FeatureSelector:                     &features.FeatureSelector{},
		MaxEdgeAddrRetries:                  defaultMaxEdgeAddrRetries,
		RPCTimeout:                          defaultRPCTimeout,
		WriteStreamTimeout:                  0,
		DisableQUICPathMTUDiscovery:         false,
		QUICConnectionLevelFlowControlLimit: defaultQUICConnFlowLimit,
		QUICStreamLevelFlowControlLimit:     defaultQUICStreamFlowLimit,
		ICMPRouterServer:                    nil,
	}, nil
}

// setupProtocolConfig configures the protocol selector and TLS configurations
func setupProtocolConfig(log *zerolog.Logger) (connection.ProtocolSelector, map[connection.Protocol]*tls.Config, error) {
	protocolSelector, err := connection.NewProtocolSelector(
		connection.HTTP2.String(),
		"random value",
		false,
		false,
		edgediscovery.ProtocolPercentage,
		connection.ResolveTTL,
		log,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create protocol selector")
	}

	edgeTLSConfigs := make(map[connection.Protocol]*tls.Config, len(connection.ProtocolList))
	for _, p := range connection.ProtocolList {
		tlsSettings := p.TLSSettings()
		if tlsSettings == nil {
			return nil, nil, errors.New("missing TLS settings for protocol")
		}

		edgeTLSConfig, err := tlsconfig.CreateTunnelConfig(
			cli.NewContext(cli.NewApp(), &flag.FlagSet{}, nil),
			tlsSettings.ServerName,
		)
		if err != nil {
			return nil, nil, errors.Wrap(err, "create edge TLS config")
		}

		if len(tlsSettings.NextProtos) > 0 {
			edgeTLSConfig.NextProtos = tlsSettings.NextProtos
		}
		edgeTLSConfigs[p] = edgeTLSConfig
	}

	return protocolSelector, edgeTLSConfigs, nil
}

// QuickTunnelResponse represents the API response for quick tunnel creation
type QuickTunnelResponse struct {
	Success bool               `json:"success"`
	Result  QuickTunnel        `json:"result"`
	Errors  []QuickTunnelError `json:"errors"`
}

// QuickTunnelError represents an error in the quick tunnel API response
type QuickTunnelError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// QuickTunnel represents a quick tunnel configuration
type QuickTunnel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname"`
	AccountTag string `json:"account_tag"`
	Secret     []byte `json:"secret"`
}

// createQuickTunnel creates a new quick tunnel using the Cloudflare API
func createQuickTunnel() (string, *connection.TunnelProperties, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   defaultHTTPTimeout,
			ResponseHeaderTimeout: defaultHTTPTimeout,
		},
		Timeout: defaultHTTPTimeout,
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tunnel", quickTunnelAPIEndpoint), nil)
	if err != nil {
		return "", nil, &TunnelError{Op: "create_request", Err: err}
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", bInfo.UserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, &TunnelError{Op: "api_request", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, &TunnelError{Op: "read_response", Err: err}
	}

	var data QuickTunnelResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return "", nil, &TunnelError{Op: "parse_response", Err: err}
	}

	if !data.Success {
		return "", nil, &TunnelError{
			Op:  "api_error",
			Err: fmt.Errorf("quick tunnel creation failed: %v", data.Errors),
		}
	}

	tunnelID, err := uuid.Parse(data.Result.ID)
	if err != nil {
		return "", nil, &TunnelError{Op: "parse_tunnel_id", Err: err}
	}

	credentials := connection.Credentials{
		AccountTag:   data.Result.AccountTag,
		TunnelSecret: data.Result.Secret,
		TunnelID:     tunnelID,
	}

	url := data.Result.Hostname
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return url, &connection.TunnelProperties{
		Credentials:    credentials,
		QuickTunnelUrl: data.Result.Hostname,
	}, nil
}

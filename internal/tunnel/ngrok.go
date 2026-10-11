package tunnel

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"golang.ngrok.com/ngrok/v2"

	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
)

// Ngrok manages an outbound tunnel to the ngrok cloud service.
type Ngrok struct {
	mu          sync.Mutex
	agent       ngrok.Agent
	fwd         ngrok.EndpointForwarder
	authToken   string
	url         string
	log         *logger.Logger
	upstreamTLS *tls.Config
}

var _ Tunneler = (*Ngrok)(nil) // compile-time interface assertion

// NgrokOption is a functional option for the [Ngrok] instance.
type NgrokOption func(*Ngrok)

// WithNgrokLogger sets the logger for the [Ngrok] instance.
func WithNgrokLogger(l *logger.Logger) NgrokOption {
	return func(n *Ngrok) { n.log = l }
}

// WithNgrokURL sets the public URL for the [Ngrok] instance.
func WithNgrokURL(url string) NgrokOption {
	return func(n *Ngrok) { n.url = url }
}

// WithNgrokTLSUpstream configures the [Ngrok] instance to forward to an HTTPS upstream using the provided TLS
// client configuration. Pass [tls.Config] with InsecureSkipVerify set to true for self-signed certificates.
func WithNgrokTLSUpstream(cfg *tls.Config) NgrokOption {
	return func(n *Ngrok) { n.upstreamTLS = cfg }
}

// NewNgrok creates a new [Ngrok] instance with the given auth token and options.
func NewNgrok(authToken string, opts ...NgrokOption) *Ngrok {
	n := &Ngrok{authToken: authToken}

	for _, opt := range opts {
		opt(n)
	}

	return n
}

// Expose starts a ngrok tunnel to the given local port and returns the public URL.
func (n *Ngrok) Expose(ctx context.Context, localPort uint16) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.fwd != nil {
		return "", ErrAlreadyStarted
	}

	var (
		agentOpts = []ngrok.AgentOption{
			ngrok.WithAuthtoken(n.authToken),
			ngrok.WithAutoConnect(true),
		}
		upstreamOpts []ngrok.UpstreamOption
		forwardsOpts []ngrok.EndpointOption
		scheme       = "http"
	)

	if n.log != nil {
		agentOpts = append(agentOpts,
			ngrok.WithLogger(n.log.Slog()),
			ngrok.WithEventHandler(n.newEventsHandler()),
		)
	}

	// setup upstream options based on TLS configuration, if provided
	if n.upstreamTLS != nil {
		scheme, upstreamOpts = "https", append(upstreamOpts, ngrok.WithUpstreamTLSClientConfig(n.upstreamTLS))
	}

	// configure "public" URL if specified, otherwise ngrok will generate it itself
	if n.url != "" {
		forwardsOpts = append(forwardsOpts, ngrok.WithURL(n.url))
	}

	// create agent
	agent, agentErr := ngrok.NewAgent(agentOpts...)
	if agentErr != nil {
		return "", fmt.Errorf("create ngrok agent: %w", agentErr)
	}

	// and forwarder
	fwd, fwdErr := agent.Forward(ctx, ngrok.WithUpstream(
		fmt.Sprintf("%s://localhost:%d", scheme, localPort), // not `127.0.0.1` for IPv6 support
		upstreamOpts...,
	), forwardsOpts...)
	if fwdErr != nil {
		if err := agent.Disconnect(); err != nil {
			return "", fmt.Errorf("start ngrok tunnel: %w; additionally, failed to disconnect agent: %w", fwdErr, err)
		}

		return "", fmt.Errorf("start ngrok tunnel: %w", fwdErr)
	}

	n.agent, n.fwd = agent, fwd

	return fwd.URL().String(), nil
}

// Close stops the active ngrok tunnel and disconnects the agent.
func (n *Ngrok) Close() error {
	n.mu.Lock()
	agent, fwd := n.agent, n.fwd
	n.agent, n.fwd = nil, nil
	n.mu.Unlock()

	if fwd == nil || agent == nil {
		return nil
	}

	closeErr := fwd.Close()
	disErr := agent.Disconnect()

	if closeErr != nil && disErr != nil {
		return fmt.Errorf("close ngrok tunnel: %w; additionally, failed to disconnect agent: %w", closeErr, disErr)
	} else if closeErr != nil {
		return fmt.Errorf("close ngrok tunnel: %w", closeErr)
	} else if disErr != nil {
		return fmt.Errorf("disconnect ngrok agent: %w", disErr)
	}

	return nil
}

func (n *Ngrok) newEventsHandler() ngrok.EventHandler {
	if n.log == nil {
		return func(e ngrok.Event) {} // no-op if no logger configured
	}

	return func(e ngrok.Event) {
		switch v := e.(type) {
		case *ngrok.EventAgentConnectSucceeded:
			n.log.Info("ngrok agent connected")
		case *ngrok.EventAgentDisconnected:
			if v.Error != nil {
				n.log.Error("ngrok agent disconnected", logger.Error(v.Error))
			} else {
				n.log.Info("ngrok agent disconnected")
			}
		case *ngrok.EventAgentHeartbeatReceived:
			n.log.Debug("ngrok heartbeat", logger.Duration("latency", v.Latency))
		case *ngrok.EventConnectionOpened:
			n.log.Debug("ngrok connection opened", logger.String("remote_addr", v.RemoteAddr))
		default:
			n.log.Debug("ngrok event", logger.String("type", e.EventType().String()))
		}
	}
}

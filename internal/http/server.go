package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/web"
)

type Server struct {
	http *http.Server

	ShutdownTimeout time.Duration // Maximum amount of time to wait for the server to stop, default is 5 seconds
}

type ServerOption func(*Server)

func WithReadTimeout(d time.Duration) ServerOption {
	return func(s *Server) { s.http.ReadTimeout = d }
}

func WithWriteTimeout(d time.Duration) ServerOption {
	return func(s *Server) { s.http.WriteTimeout = d }
}

func WithIDLETimeout(d time.Duration) ServerOption {
	return func(s *Server) { s.http.IdleTimeout = d }
}

func NewServer(baseCtx context.Context, log *zap.Logger, opts ...ServerOption) *Server {
	var (
		server = Server{
			http: &http.Server{ //nolint:gosec
				BaseContext: func(net.Listener) context.Context { return baseCtx },
				ErrorLog:    zap.NewStdLog(log),
			},
			ShutdownTimeout: 5 * time.Second, //nolint:mnd
		}
	)

	for _, opt := range opts {
		opt(&server)
	}

	return &server
}

func (s *Server) Register(ctx context.Context, log *zap.Logger, useLiveFrontend bool) *Server {
	var frontendFs = web.Dist(useLiveFrontend)

	_ = frontendFs // FIXME

	var (
		// create openapi server implementation
		openapiServer = NewOpenAPI(ctx, log)

		// create the base router for the openapi server
		openapiMux = http.NewServeMux()

		// "convert" the openapi server to the [http.Handler]
		openapiHandler = openapi.HandlerWithOptions(openapiServer, openapi.StdHTTPServerOptions{
			ErrorHandlerFunc: openapiServer.HandleInternalError,
			BaseRouter:       openapiMux,
			Middlewares:      []openapi.MiddlewareFunc{openapi.CorsMiddleware()},
		})
	)

	_ = openapiHandler // FIXME

	return s
}

// StartHTTP starts the HTTP server. It listens on the provided listener and serves incoming requests.
// To stop the server, cancel the provided context.
//
// It blocks until the context is canceled or the server is stopped by some error.
func (s *Server) StartHTTP(ctx context.Context, ln net.Listener) error {
	var errCh = make(chan error)

	go func(ch chan<- error) { defer close(ch); ch <- s.http.Serve(ln) }(errCh)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.ShutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err, isOpened := <-errCh:
		switch {
		case !isOpened:
			return nil
		case err != nil:
			return err
		}
	}

	return nil
}

package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/frontend"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/middleware/logreq"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/middleware/webhook"
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
	var (
		oAPI    = NewOpenAPI(ctx, log)                    // OpenAPI server implementation
		spa     = frontend.New(web.Dist(useLiveFrontend)) // file server for SPA (also handles 404 errors)
		mux     = http.NewServeMux()                      // base router for the OpenAPI server
		handler = openapi.HandlerWithOptions(oAPI, openapi.StdHTTPServerOptions{
			ErrorHandlerFunc: oAPI.HandleInternalError, // set error handler for internal server errors
			BaseRouter:       mux,
		})
	)

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// custom logic for handling 404 errors
		if strings.HasPrefix(strings.TrimLeft(r.URL.Path, "/"), "api") {
			// if the request path starts with "api", return the 404 error in the format required by the API
			oAPI.HandleNotFoundError(w, r)
		} else {
			// otherwise, serve the SPA frontend
			spa.ServeHTTP(w, r)
		}
	}))

	// apply middlewares
	s.http.Handler = logreq.New(log, nil)( // logger middleware
		webhook.New(log)( // webhook capture as a middleware
			handler,
		),
	)

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

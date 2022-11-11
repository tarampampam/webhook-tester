package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/tarampampam/webhook-tester/internal/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type (
	Server struct {
		log    *zap.Logger
		server *http.Server
		router *mux.Router
	}
)

const (
	readTimeout = time.Second * 5

	// IMPORTANT! Must be grater then
	// github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/create.maxResponseDelay value!
	writeTimeout = time.Second * 31
)

// NewServer creates new server instance.
func NewServer(log *zap.Logger) *Server {
	var (
		router     = mux.NewRouter()
		httpServer = &http.Server{
			Handler:           router,
			ErrorLog:          zap.NewStdLog(log),
			ReadHeaderTimeout: readTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
		}
	)

	return &Server{
		log:    log,
		server: httpServer,
		router: router,
	}
}

// Start server.
func (s *Server) Start(ip string, port uint16) error {
	s.server.Addr = ip + ":" + strconv.Itoa(int(port))

	return s.server.ListenAndServe()
}

// Register server routes, middlewares, etc.
func (s *Server) Register(
	ctx context.Context,
	cfg config.Config,
	rdb *redis.Client,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
) error {
	registry := metrics.NewRegistry()

	s.registerGlobalMiddlewares()

	if err := s.registerHandlers(ctx, cfg, stor, pub, sub, rdb, registry); err != nil {
		return err
	}

	return nil
}

func (s *Server) registerGlobalMiddlewares() {
	s.router.Use(
		logreq.New(s.log),
		panic.New(s.log),
	)
}

// registerHandlers register server http handlers.
func (s *Server) registerHandlers(
	ctx context.Context,
	cfg config.Config,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
	rdb *redis.Client,
	registry *prometheus.Registry,
) error {
	if err := s.registerWebHookHandlers(ctx, cfg, stor, pub, registry); err != nil {
		return err
	}

	s.registerAPIHandlers(cfg, stor, pub)

	if err := s.registerWebsocketHandlers(ctx, cfg, stor, pub, sub, registry); err != nil {
		return err
	}

	s.registerServiceHandlers(ctx, rdb, registry)

	if err := s.registerFileServerHandler(); err != nil {
		return err
	}

	return nil
}

// Stop server.
func (s *Server) Stop(ctx context.Context) error { return s.server.Shutdown(ctx) }

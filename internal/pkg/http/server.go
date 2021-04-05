package http

import (
	"context"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"go.uber.org/zap"
)

type (
	Server struct {
		log    *zap.Logger
		server *http.Server
		router *mux.Router
	}
)

const (
	defaultReadTimeout  = time.Second * 5
	defaultWriteTimeout = time.Second * 15
)

// NewServer creates new server instance.
func NewServer(log *zap.Logger) *Server {
	var (
		router     = mux.NewRouter()
		httpServer = &http.Server{
			Handler:      router,
			ErrorLog:     zap.NewStdLog(log),
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout, // TODO check with large webhook response delay
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
	publicDir string,
	rdb *redis.Client,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
) error {
	s.registerGlobalMiddlewares()

	if err := s.registerHandlers(ctx, cfg, stor, pub, sub, publicDir, rdb); err != nil {
		return err
	}

	if err := s.registerCustomMimeTypes(); err != nil {
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
	publicDir string,
	rdb *redis.Client,
) error {
	if err := s.registerWebHookHandlers(ctx, cfg, stor, pub); err != nil {
		return err
	}

	s.registerAPIHandlers(cfg, stor, pub)
	s.registerWebsocketHandlers(ctx, cfg, stor, pub, sub, s.log)
	s.registerServiceHandlers(ctx, rdb)

	if publicDir != "" {
		if err := s.registerFileServerHandler(publicDir); err != nil {
			return err
		}
	}

	return nil
}

// registerCustomMimeTypes registers custom mime types.
func (*Server) registerCustomMimeTypes() error {
	return mime.AddExtensionType(".vue", "text/html; charset=utf-8") // this is my whim :)
}

// Stop server.
func (s *Server) Stop(ctx context.Context) error { return s.server.Shutdown(ctx) }

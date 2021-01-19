package http

import (
	"context"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"go.uber.org/zap"
)

type (
	Server struct {
		ctx         context.Context
		log         *zap.Logger
		publicDir   string
		appSettings *settings.AppSettings
		server      *http.Server
		router      *mux.Router
		storage     storage.Storage
		broadcaster broadcaster
		rdb         *redis.Client
	}
)

type broadcaster interface {
	Publish(channel string, event broadcast.Event) error
}

const (
	defaultReadTimeout  = time.Second * 5
	defaultWriteTimeout = time.Second * 15
)

// NewServer creates new server instance.
func NewServer(
	ctx context.Context,
	log *zap.Logger,
	publicDir string,
	appSettings *settings.AppSettings,
	storage storage.Storage,
	br broadcaster,
	rdb *redis.Client,
) *Server {
	var (
		router     = mux.NewRouter()
		httpServer = &http.Server{
			Handler:      router,
			ErrorLog:     zap.NewStdLog(log),
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout,
		}
	)

	return &Server{
		ctx:         ctx,
		log:         log,
		publicDir:   publicDir,
		appSettings: appSettings,
		server:      httpServer,
		router:      router,
		storage:     storage,
		broadcaster: br,
		rdb:         rdb,
	}
}

// Start server.
func (s *Server) Start(ip string, port uint16) error {
	s.server.Addr = ip + ":" + strconv.Itoa(int(port))

	return s.server.ListenAndServe()
}

// Register server routes, middlewares, etc.
func (s *Server) Register() error {
	s.registerGlobalMiddlewares()

	if err := s.registerHandlers(); err != nil {
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
func (s *Server) registerHandlers() error {
	if err := s.registerWebHookHandlers(); err != nil {
		return err
	}

	s.registerAPIHandlers()
	s.registerServiceHandlers()

	if s.publicDir != "" {
		if err := s.registerFileServerHandler(s.publicDir); err != nil {
			return err
		}
	}

	return nil
}

// registerCustomMimeTypes registers custom mime types.
func (*Server) registerCustomMimeTypes() error {
	return mime.AddExtensionType(".vue", "text/html; charset=utf-8")
}

// Stop server.
func (s *Server) Stop(ctx context.Context) error { return s.server.Shutdown(ctx) }

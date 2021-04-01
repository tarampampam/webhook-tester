package http

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"go.uber.org/zap"
)

type broadcaster interface {
	Publish(channel string, event broadcast.Event) error
}

type (
	Server struct {
		log    *zap.Logger
		server *http.Server
		router *mux.Router

		afterShutdown []func()
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
		log:           log,
		server:        httpServer,
		router:        router,
		afterShutdown: make([]func(), 0, 1),
	}
}

func (s *Server) addOnShutdown(f func()) { s.afterShutdown = append(s.afterShutdown, f) }

// Start server.
func (s *Server) Start(ip string, port uint16) error {
	s.server.Addr = ip + ":" + strconv.Itoa(int(port))

	return s.server.ListenAndServe()
}

// Register server routes, middlewares, etc.
func (s *Server) Register(ctx context.Context, cfg config.Config, publicDir string, rdb *redis.Client) error {
	var br broadcaster

	switch cfg.BroadcastDriver {
	case config.BroadcastDriverPusher:
		br = broadcast.NewPusher(cfg.Pusher.AppID, cfg.Pusher.Key, cfg.Pusher.Secret, cfg.Pusher.Cluster)
	case config.BroadcastDriverNone:
		br = &broadcast.None{}
	default:
		return errors.New("unsupported broadcast driver")
	}

	var stor storage.Storage

	switch cfg.StorageDriver {
	case config.StorageDriverRedis:
		stor = storage.NewRedisStorage(ctx, rdb, cfg.SessionTTL, cfg.MaxRequests)
	case config.StorageDriverMemory:
		inmemory := storage.NewInMemoryStorage(cfg.SessionTTL, cfg.MaxRequests)

		s.addOnShutdown(func() { _ = inmemory.Close() })

		stor = inmemory
	default:
		return errors.New("unsupported storage driver")
	}

	s.registerGlobalMiddlewares()

	if err := s.registerHandlers(ctx, cfg, stor, br, publicDir, rdb); err != nil {
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
	storage storage.Storage,
	br broadcaster,
	publicDir string,
	rdb *redis.Client,
) error {
	if err := s.registerWebHookHandlers(ctx, cfg, storage, br); err != nil {
		return err
	}

	s.registerAPIHandlers(cfg, storage, br)
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
	return mime.AddExtensionType(".vue", "text/html; charset=utf-8")
}

// Stop server.
func (s *Server) Stop(ctx context.Context) error {
	defer func() {
		for i := 0; i < len(s.afterShutdown); i++ {
			s.afterShutdown[i]()
		}
	}()

	return s.server.Shutdown(ctx)
}

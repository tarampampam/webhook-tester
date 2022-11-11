package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/metrics"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

const (
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 31 // IMPORTANT! Must be grater then create.maxResponseDelay value!
)

type Server struct {
	log *zap.Logger
	srv *http.Server
}

func NewServer(log *zap.Logger) *Server {
	return &Server{
		log: log,
		srv: &http.Server{
			ErrorLog:          zap.NewStdLog(log),
			ReadHeaderTimeout: readTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
		},
	}
}

func (s *Server) Register(
	ctx context.Context,
	cfg config.Config,
	rdb *redis.Client,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
) {
	registry := metrics.NewRegistry()

	s.srv.Handler = api.HandlerWithOptions(&API{
		ctx:  ctx,
		cfg:  cfg,
		rdb:  rdb,
		stor: stor,
		pub:  pub,
		sub:  sub,
		reg:  registry,
	}, api.GorillaServerOptions{
		Middlewares: []api.MiddlewareFunc{
			logreq.New(s.log),
			panic.New(s.log),
		},
	})
}

// Start the server.
func (s *Server) Start(ip string, port uint16) error {
	s.srv.Addr = ip + ":" + strconv.Itoa(int(port))

	return s.srv.ListenAndServe()
}

// Stop the server.
func (s *Server) Stop(ctx context.Context) error { return s.srv.Shutdown(ctx) }

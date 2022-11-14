package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/http/fileserver"
	apiHandlers "github.com/tarampampam/webhook-tester/internal/http/handlers"
	"github.com/tarampampam/webhook-tester/internal/http/middlewares/logreq"
	"github.com/tarampampam/webhook-tester/internal/http/middlewares/panic"
	"github.com/tarampampam/webhook-tester/internal/http/middlewares/webhook"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/metrics"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"github.com/tarampampam/webhook-tester/internal/pkg/version"
	"github.com/tarampampam/webhook-tester/web"
)

const (
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 31 // IMPORTANT! Must be grater then create.maxResponseDelay value!
)

type Server struct {
	log  *zap.Logger
	echo *echo.Echo
}

func NewServer(log *zap.Logger) *Server {
	var srv = echo.New()

	srv.StdLogger = zap.NewStdLog(log)
	srv.Server.ReadTimeout = readTimeout
	srv.Server.ReadHeaderTimeout = readTimeout
	srv.Server.WriteTimeout = writeTimeout
	srv.Server.ErrorLog = srv.StdLogger
	srv.IPExtractor = NewIPExtractor()
	srv.HideBanner = true
	srv.HidePort = true

	return &Server{
		log:  log,
		echo: srv,
	}
}

func (s *Server) Register(
	ctx context.Context,
	cfg config.Config,
	rdb *redis.Client,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
) error {
	registry := metrics.NewRegistry()

	s.echo.Use(
		logreq.New(s.log, []string{"/ready", "/health"}),
		panic.New(s.log),
	)

	websocketMetrics := metrics.NewWebsockets()
	if err := websocketMetrics.Register(registry); err != nil {
		return err
	}

	api.RegisterHandlers(s.echo, apiHandlers.NewAPI(
		ctx,
		cfg,
		rdb,
		stor,
		pub,
		sub,
		registry,
		version.Version(),
		&websocketMetrics,
	))

	webhookMetrics := metrics.NewWebhooks()
	if err := webhookMetrics.Register(registry); err != nil {
		return err
	}

	var (
		wh     = webhook.New(ctx, cfg, stor, pub, &webhookMetrics)
		static = fileserver.NewHandler(http.FS(web.Content()))
	)

	s.echo.Any("/*", wh(func(c echo.Context) error { // wrap file server into webhook middleware
		if method := c.Request().Method; method == http.MethodGet || method == http.MethodHead {
			return static(c)
		}

		s.echo.HTTPErrorHandler(echo.ErrNotFound, c)

		return nil
	}))

	return nil
}

// Start the server.
func (s *Server) Start(ip string, port uint16) error {
	return s.echo.Start(ip + ":" + strconv.Itoa(int(port)))
}

// Stop the server.
func (s *Server) Stop(ctx context.Context) error { return s.echo.Shutdown(ctx) }

package http

import (
	"context"
	"net/http"

	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"

	"github.com/go-redis/redis/v8"
	"github.com/tarampampam/webhook-tester/internal/pkg/checkers"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/fileserver"
	apiSessionCreate "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/create"
	sessionDelete "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/delete"
	getAllRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/all"
	clearRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/clear"
	deleteRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/delete"
	getRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/get"
	apiSettings "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/settings"
	apiVersion "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/version"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/healthz"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/webhook"
	websocketSession "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/websocket/session"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/cors"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/json"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/middlewares/nocache"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	"github.com/tarampampam/webhook-tester/internal/pkg/version"
	"go.uber.org/zap"
)

const uuidPattern string = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

func (s *Server) registerWebHookHandlers(
	ctx context.Context,
	cfg config.Config,
	storage storage.Storage,
	pub pubsub.Publisher,
) error {
	allowedMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodTrace,
	}

	webhookRouter := s.router.
		PathPrefix("").
		Subrouter()

	webhookRouter.Use(cors.New())

	handler := webhook.NewHandler(ctx, cfg, storage, pub) // TODO return error if wrong config passed

	webhookRouter.
		Handle("/{sessionUUID:"+uuidPattern+"}", handler).
		Methods(allowedMethods...).
		Name("webhook")

	webhookRouter.
		Handle("/{sessionUUID:"+uuidPattern+"}/{statusCode:[1-5][0-9][0-9]}", handler).
		Methods(allowedMethods...).
		Name("webhook_with_status_code")

	webhookRouter.
		Handle("/{sessionUUID:"+uuidPattern+"}/{any:.*}", handler).
		Methods(allowedMethods...).
		Name("webhook_any")

	return nil
}

func (s *Server) registerAPIHandlers(cfg config.Config, storage storage.Storage, pub pubsub.Publisher) {
	apiRouter := s.router.
		PathPrefix("/api").
		Subrouter()

	apiRouter.Use(nocache.New(), json.New())

	// get application settings
	apiRouter.
		HandleFunc("/settings", apiSettings.NewGetSettingsHandler(cfg)).
		Methods(http.MethodGet).
		Name("api_settings_get")

	apiRouter.
		HandleFunc("/version", apiVersion.NewHandler(version.Version())).
		Methods(http.MethodGet).
		Name("api_get_version")

	// create new session
	apiRouter.
		HandleFunc("/session", apiSessionCreate.NewHandler(storage)).
		Methods(http.MethodPost).
		Name("api_session_create")

	// delete session with passed UUID
	apiRouter.
		HandleFunc("/session/{sessionUUID:"+uuidPattern+"}", sessionDelete.NewHandler(storage)).
		Methods(http.MethodDelete).
		Name("api_session_delete")

	// get requests list for session with passed UUID
	apiRouter.
		HandleFunc("/session/{sessionUUID:"+uuidPattern+"}/requests", getAllRequests.NewHandler(storage)).
		Methods(http.MethodGet).
		Name("api_session_requests_all_get")

	// get request details by UUID for session with passed UUID
	apiRouter.
		HandleFunc(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			getRequest.NewHandler(storage),
		).
		Methods(http.MethodGet).
		Name("api_session_request_get")

	// delete request by UUID for session with passed UUID
	apiRouter.
		HandleFunc(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			deleteRequest.NewHandler(storage, pub),
		).
		Methods(http.MethodDelete).
		Name("api_delete_session_request")

	// delete all requests for session with passed UUID
	apiRouter.
		HandleFunc("/session/{sessionUUID:"+uuidPattern+"}/requests", clearRequests.NewHandler(storage, pub)).
		Methods(http.MethodDelete).
		Name("api_delete_all_session_requests")
}

func (s *Server) registerWebsocketHandlers(
	ctx context.Context,
	cfg config.Config,
	storage storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
	log *zap.Logger,
) {
	wsRouter := s.router.
		PathPrefix("/ws").
		Subrouter()

	wsRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}", websocketSession.NewHandler(ctx, cfg, storage, pub, sub, log)).
		Methods(http.MethodGet).
		Name("ws_session")
}

func (s *Server) registerServiceHandlers(ctx context.Context, rdb *redis.Client) {
	s.router.
		HandleFunc("/ready", healthz.NewHandler(checkers.NewReadyChecker(ctx, rdb))).
		Methods(http.MethodGet, http.MethodHead).
		Name("ready")

	s.router.
		HandleFunc("/live", healthz.NewHandler(checkers.NewLiveChecker())).
		Methods(http.MethodGet, http.MethodHead).
		Name("live")

	// TODO add "/uptime" handler
	// TODO add "/metrics" handler
}

func (s *Server) registerFileServerHandler(publicDir string) error {
	fs, err := fileserver.NewFileServer(fileserver.Settings{
		FilesRoot:               publicDir,
		IndexFileName:           "index.html",
		ErrorFileName:           "__error__.html",
		RedirectIndexFileToRoot: true,
	})
	if err != nil {
		return err
	}

	s.router.
		PathPrefix("/").
		Methods(http.MethodGet, http.MethodHead).
		Handler(fs).
		Name("static")

	return nil
}

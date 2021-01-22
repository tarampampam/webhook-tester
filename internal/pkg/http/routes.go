package http

import (
	"context"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/tarampampam/webhook-tester/internal/pkg/checkers"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	sessionCreate "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/create"
	sessionDelete "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/delete"
	getAllRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/all"
	clearRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/clear"
	deleteRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/delete"
	getRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/get"
	settingsGet "github.com/tarampampam/webhook-tester/internal/pkg/http/api/settings/get"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/fileserver"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/healthz"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/webhook"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

const uuidPattern string = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

func (s *Server) registerWebHookHandlers(cfg config.Config, storage storage.Storage, br broadcaster) error {
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

	webhookRouter.Use(AllowCORSMiddleware)

	handler := webhook.NewHandler(cfg, storage, br) // TODO return error if wrong config passed

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

func (s *Server) registerAPIHandlers(cfg config.Config, storage storage.Storage, br broadcaster) {
	apiRouter := s.router.
		PathPrefix("/api").
		Subrouter()

	apiRouter.Use(DisableCachingMiddleware, JSONResponseMiddleware)

	// get application settings
	apiRouter.
		Handle("/settings", settingsGet.NewHandler(cfg)). // FIXME settings passed using pointer, it is really needed?
		Methods(http.MethodGet).
		Name("api_settings_get")

	// create new session
	apiRouter.
		Handle("/session", sessionCreate.NewHandler(storage)).
		Methods(http.MethodPost).
		Name("api_session_create")

	// delete session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}", sessionDelete.NewHandler(storage)).
		Methods(http.MethodDelete).
		Name("api_session_delete")

	// get requests list for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", getAllRequests.NewHandler(storage)).
		Methods(http.MethodGet).
		Name("api_session_requests_all_get")

	// get request details by UUID for session with passed UUID
	apiRouter.
		Handle(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			getRequest.NewHandler(storage),
		).
		Methods(http.MethodGet).
		Name("api_session_request_get")

	// delete request by UUID for session with passed UUID
	apiRouter.
		Handle(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			deleteRequest.NewHandler(storage, br),
		).
		Methods(http.MethodDelete).
		Name("api_delete_session_request")

	// delete all requests for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", clearRequests.NewHandler(storage, br)).
		Methods(http.MethodDelete).
		Name("api_delete_all_session_requests")
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

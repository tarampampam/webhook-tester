package http

import (
	"net/http"

	sessionCreate "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/create"
	sessionDelete "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/delete"
	getAllRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/all"
	clearRequests "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/clear"
	deleteRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/delete"
	getRequest "github.com/tarampampam/webhook-tester/internal/pkg/http/api/session/requests/get"
	settingsGet "github.com/tarampampam/webhook-tester/internal/pkg/http/api/settings/get"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/fileserver"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/probes"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/webhook"
)

const uuidPattern string = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

// RegisterHandlers register server http handlers.
func (s *Server) RegisterHandlers() {
	s.registerServiceHandlers()
	s.registerAPIHandlers()
	s.registerWebHookHandlers()
	s.registerFileServerHandler()
}

// Register "service" handlers.
func (s *Server) registerServiceHandlers() {
	// liveness probe
	s.Router.
		Handle("/live", probes.NewLivenessHandler()).
		Methods(http.MethodGet).
		Name("liveness_probe")

	// readiness probe
	s.Router.
		Handle("/ready", probes.NewReadinessHandler(s.storage)).
		Methods(http.MethodGet).
		Name("readiness_probe")
}

// Register API handlers.
func (s *Server) registerAPIHandlers() {
	apiRouter := s.Router.
		PathPrefix("/api").
		Subrouter()

	apiRouter.Use(DisableCachingMiddleware, JSONResponseMiddleware)

	// get application settings
	apiRouter.
		Handle("/settings", settingsGet.NewHandler(s.appSettings)).
		Methods(http.MethodGet).
		Name("api_settings_get")

	// create new session
	apiRouter.
		Handle("/session", sessionCreate.NewHandler(s.storage)).
		Methods(http.MethodPost).
		Name("api_session_create")

	// delete session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}", sessionDelete.NewHandler(s.storage)).
		Methods(http.MethodDelete).
		Name("api_session_delete")

	// get requests list for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", getAllRequests.NewHandler(s.storage)).
		Methods(http.MethodGet).
		Name("api_session_requests_all_get")

	// get request details by UUID for session with passed UUID
	apiRouter.
		Handle(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			getRequest.NewHandler(s.storage),
		).
		Methods(http.MethodGet).
		Name("api_session_request_get")

	// delete request by UUID for session with passed UUID
	apiRouter.
		Handle(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			deleteRequest.NewHandler(s.storage, s.broadcaster),
		).
		Methods(http.MethodDelete).
		Name("api_delete_session_request")

	// delete all requests for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", clearRequests.NewHandler(s.storage, s.broadcaster)).
		Methods(http.MethodDelete).
		Name("api_delete_all_session_requests")
}

// Register incoming webhook handlers.
func (s *Server) registerWebHookHandlers() {
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

	webhookRouter := s.Router.
		PathPrefix("").
		Subrouter()

	webhookRouter.Use(AllowCORSMiddleware)

	handler := webhook.NewHandler(s.appSettings, s.storage, s.broadcaster)

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
}

// Register file server handler.
func (s *Server) registerFileServerHandler() {
	s.Router.
		PathPrefix("/").
		Handler(fileserver.NewFileServer(fileserver.Settings{
			Root:         http.Dir(s.settings.PublicAssetsDirectoryPath),
			IndexFile:    "index.html",
			Error404file: "404.html",
		})).
		Name("static")
}

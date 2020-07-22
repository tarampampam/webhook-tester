package http

import (
	"net/http"
	sessionCreate "webhook-tester/http/api/session/create"
	sessionDelete "webhook-tester/http/api/session/delete"
	settingsGet "webhook-tester/http/api/settings/get"
	"webhook-tester/http/fileserver"
	"webhook-tester/http/ping"
	"webhook-tester/http/stub"
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
	// just a ping, no more
	s.Router.
		Handle("/ping", DisableCachingMiddleware(ping.NewHandler())).
		Methods(http.MethodGet).
		Name("ping")
}

// Register API handlers.
func (s *Server) registerAPIHandlers() { //nolint:funlen
	apiRouter := s.Router.
		PathPrefix("/api").
		Subrouter()

	apiRouter.Use(DisableCachingMiddleware, JSONResponseMiddleware)

	// get application settings
	apiRouter.
		Handle("/settings", settingsGet.NewHandler(s.appSettings)).
		Methods(http.MethodGet).
		Name("settings_get")

	// create new session
	apiRouter.
		Handle("/session", sessionCreate.NewHandler(s.appSettings, s.storage)).
		Methods(http.MethodPost).
		Name("session_create")

	// delete session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}", sessionDelete.NewHandler(s.storage)).
		Methods(http.MethodDelete).
		Name("session_delete")

	// get requests list for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", stub.Handler(`{
			"11111111-0000-0000-0000-000000000000": {
				"client_address": "1.1.1.1",
				"method": "GET",
				"content": "fake content goes here<\/code><\/pre><script>alert(1)<\/script>",
				"headers": {
					"host": "foo.example.com",
					"user-agent": "curl\/7.58.0",
					"accept": "text\/html,application\/xhtml+xml",
					"accept-encoding": "gzip",
					"accept-language": "en,ru;q=0.9",
					"cdn-loop": "cloudflare",
					"cf-connecting-ip": "111.111.111.111",
					"cookie": "__cfduid=d0bca19992c54486ae9372d7d4d3096531595016640",
					"dnt": "1"
				},
				"url": "https://foo.example.com/%RAND_UUID%/foobar",
				"created_at_unix": 1595017226
			},
			"22222222-0000-0000-0000-000000000000": {
				"client_address": "1.1.1.1",
				"method": "PUT",
				"content": "{\"foo\":1,\"bar\":\"baz\",\"a\":[1,2,3]}",
				"headers": {
					"host": "foo.example.com",
					"user-agent": "curl\/7.58.0",
					"accept": "text\/html,application\/xhtml+xml",
					"accept-encoding": "gzip",
					"accept-language": "en,ru;q=0.9",
					"cdn-loop": "cloudflare",
					"cf-connecting-ip": "111.111.111.111",
					"cookie": "__cfduid=d0bca19992c54486ae9372d7d4d3096531595016640",
					"dnt": "1"
				},
				"url": "https://foo.example.com/%RAND_UUID%/barbaz",
				"created_at_unix": 1595017240
			},
			"33333333-0000-0000-0000-000000000000": {
				"client_address": "2.2.2.2",
				"method": "DELETE",
				"content": "",
				"headers": {
					"host": "foo.example.boo",
					"user-agent": "curl\/7.58.0",
					"accept": "text\/html,application\/xhtml+xml",
					"accept-encoding": "gzip",
					"accept-language": "en,ru;q=0.9",
					"cdn-loop": "cloudflare",
					"cf-connecting-ip": "22.22.22.22",
					"cookie": "__cfduid=d0bca19992c54486ae9372d7d4d3096531595016640",
					"dnt": "0"
				},
				"url": "https://foo.example.com/%RAND_UUID%/blah",
				"created_at_unix": 1595017340
			}
		}`)).
		Methods(http.MethodGet).
		Name("session_requests_all_get")

	// get request details by UUID for session with passed UUID
	apiRouter.
		Handle(
			"/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}",
			stub.Handler(`{
				"client_address": "1.1.1.1",
				"method": "GET",
				"content": "fake content goes here",
				"headers": {
					"host": "foo.example.com",
					"user-agent": "curl\/7.58.0",
					"accept": "text\/html,application\/xhtml+xml",
					"accept-encoding": "gzip",
					"accept-language": "en,ru;q=0.9",
					"cdn-loop": "cloudflare",
					"cf-connecting-ip": "111.111.111.111",
					"cookie": "__cfduid=d0bca19992c54486ae9372d7d4d3096531595016640",
					"dnt": "1"
				},
				"url": "https://foo.example.com/aaaaaaaa-bbbb-cccc-dddd-000000000000/foobar",
				"created_at_unix": 1595017226
			}`),
		).
		Methods(http.MethodGet).
		Name("session_request_get")

	// delete request by UUID for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests/{requestUUID:"+uuidPattern+"}", stub.Handler(`{
			"success": true
		}`)).
		Methods(http.MethodDelete).
		Name("delete_session_request")

	// delete all requests for session with passed UUID
	apiRouter.
		Handle("/session/{sessionUUID:"+uuidPattern+"}/requests", stub.Handler(`{
			"success": true
		}`)).
		Methods(http.MethodDelete).
		Name("delete_all_session_requests")
}

// Register incoming webhook handlers.
func (s *Server) registerWebHookHandlers() {
	allMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	// @todo: add CORS middleware

	s.Router.
		Handle("/{sessionUUID:"+uuidPattern+"}", stub.Handler(`"foobar"`)).
		Methods(allMethods...).
		Name("webhook")

	s.Router.
		Handle("/{sessionUUID:"+uuidPattern+"}/{statusCode:[1-5][0-9][0-9]}", stub.Handler(`"foobar"`)).
		Methods(allMethods...).
		Name("webhook_with_status_code")

	s.Router.
		Handle("/{sessionUUID:"+uuidPattern+"}/{any:.*}", stub.Handler(`"foobar"`)).
		Methods(allMethods...).
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

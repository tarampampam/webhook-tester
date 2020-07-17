package http

import (
	"net/http"
	"webhook-tester/http/errors"
	"webhook-tester/http/fileserver"
	"webhook-tester/http/ping"
)

// RegisterHandlers register server http handlers.
func (s *Server) RegisterHandlers() {
	s.registerErrorHandlers()
	s.registerAPIHandlers()
	s.registerFileServerHandler()
}

func (s *Server) registerErrorHandlers() {
	s.Router.NotFoundHandler = errors.NotFoundHandler()
	s.Router.MethodNotAllowedHandler = errors.MethodNotAllowedHandler()
}

func (s *Server) registerAPIHandlers() {
	s.Router.
		Handle("/ping", DisableCachingMiddleware(ping.NewHandler())).
		Methods(http.MethodGet).
		Name("ping")
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

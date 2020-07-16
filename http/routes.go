package http

import (
	"net/http"
	"webhook-tester/http/errors"
	"webhook-tester/http/ping"
)

// RegisterHandlers register server http handlers.
func (s *Server) RegisterHandlers() {
	s.Router.NotFoundHandler = errors.NotFoundHandler()
	s.Router.MethodNotAllowedHandler = errors.MethodNotAllowedHandler()

	s.Router.
		Handle("/ping", DisableCachingMiddleware(ping.NewHandler())).
		Methods(http.MethodGet).
		Name("ping")
}

package requests_subscribe

import (
	"context"
	"errors"
	"net/http"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type sID = openapi.SessionUUIDInPath

// Handler handles WebSocket subscriptions for real-time captured request notifications.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle upgrades the connection to WebSocket and streams request events for the given session.
func (h *Handler) Handle(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ sID) error {
	return errors.New("not implemented")
}

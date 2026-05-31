package settings_get

import "gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"

// Handler handles HTTP requests for reading application settings.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle returns the current application settings as a response.
func (h *Handler) Handle() openapi.SettingsResponse { return openapi.SettingsResponse{} }

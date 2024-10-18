package settings_get

import (
	"gh.tarampamp.am/webhook-tester/v2/internal/config"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

type Handler struct{ cfg config.AppSettings }

func New(s config.AppSettings) *Handler { return &Handler{cfg: s} }

func (h *Handler) Handle() (resp openapi.SettingsResponse) {
	resp.Limits.MaxRequestBodySize = h.cfg.MaxRequestBodySize
	resp.Limits.MaxRequests = h.cfg.MaxRequests
	resp.Limits.SessionTtl = uint32(h.cfg.SessionTTL.Seconds())

	return
}

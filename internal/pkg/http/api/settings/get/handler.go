package get

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/version"
)

type Handler struct {
	cfg  config.Config
	json jsoniter.API
}

func NewHandler(cfg config.Config) http.Handler {
	return &Handler{
		cfg:  cfg,
		json: jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = h.json.NewEncoder(w).Encode(response{
		Version: version.Version(),
		Pusher: pusher{
			Key:     h.cfg.Pusher.Key,
			Cluster: h.cfg.Pusher.Cluster,
		},
		Limits: responseLimits{
			MaxRequests:        h.cfg.MaxRequests,
			SessionLifetimeSec: uint32(h.cfg.SessionTTL.Seconds()),
		},
	})
}

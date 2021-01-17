package get

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/version"
)

type Handler struct {
	appSettings *settings.AppSettings
	json        jsoniter.API
}

func NewHandler(appSettings *settings.AppSettings) http.Handler {
	return &Handler{
		appSettings: appSettings,
		json:        jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = h.json.NewEncoder(w).Encode(response{
		Version: version.Version(),
		Pusher: pusher{
			Key:     h.appSettings.PusherKey,
			Cluster: h.appSettings.PusherCluster,
		},
		Limits: responseLimits{
			MaxRequests:        h.appSettings.MaxRequests,
			SessionLifetimeSec: uint32(h.appSettings.SessionTTL.Seconds()),
		},
	})
}

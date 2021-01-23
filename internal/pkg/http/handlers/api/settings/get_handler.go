package settings

import (
	"encoding/json"
	"net/http"

	"github.com/tarampampam/webhook-tester/internal/pkg/config"
)

func NewGetSettingsHandler(cfg config.Config) http.HandlerFunc {
	var c []byte // response in-memory cache

	return func(w http.ResponseWriter, _ *http.Request) {
		if c == nil {
			// set basic properties
			resp := struct {
				BroadcastDriver string `json:"broadcast_driver"` // TODO new property, update front
				Pusher          struct {
					Key     string `json:"key"`
					Cluster string `json:"cluster"`
				} `json:"pusher"`
				Limits struct {
					MaxRequests               uint16 `json:"max_requests"`
					MaxWebhookRequestBodySize uint32 `json:"max_webhook_body_size"` // TODO new property, update front
					SessionLifetimeSec        uint32 `json:"session_lifetime_sec"`
				} `json:"limits"`
			}{}

			resp.BroadcastDriver = cfg.BroadcastDriver.String()
			resp.Pusher.Key = cfg.Pusher.Key
			resp.Pusher.Cluster = cfg.Pusher.Cluster
			resp.Limits.MaxRequests = cfg.MaxRequests
			resp.Limits.MaxWebhookRequestBodySize = cfg.MaxRequestBodySize
			resp.Limits.SessionLifetimeSec = uint32(cfg.SessionTTL.Seconds())

			c, _ = json.Marshal(resp)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(c)
	}
}

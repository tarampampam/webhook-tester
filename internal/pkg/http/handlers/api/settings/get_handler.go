package settings

import (
	"net/http"

	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"

	jsoniter "github.com/json-iterator/go"
)

func NewGetSettingsHandler(cfg config.Config) http.HandlerFunc {
	out := settingsOutput{
		BroadcastDriver: cfg.BroadcastDriver.String(),
	}

	out.Pusher.Key = cfg.Pusher.Key
	out.Pusher.Cluster = cfg.Pusher.Cluster
	out.Limits.MaxRequests = cfg.MaxRequests
	out.Limits.MaxWebhookRequestBodySize = cfg.MaxRequestBodySize
	out.Limits.SessionLifetimeSec = uint32(cfg.SessionTTL.Seconds())

	return func(w http.ResponseWriter, _ *http.Request) {
		responder.JSON(w, out)
	}
}

type settingsOutput struct {
	BroadcastDriver string `json:"broadcast_driver"` // TODO new property, update frontend
	Pusher          struct {
		Key     string `json:"key"`
		Cluster string `json:"cluster"`
	} `json:"pusher"`
	Limits struct {
		MaxRequests               uint16 `json:"max_requests"`
		MaxWebhookRequestBodySize uint32 `json:"max_webhook_body_size"` // TODO new property, update frontend
		SessionLifetimeSec        uint32 `json:"session_lifetime_sec"`
	} `json:"limits"`
}

func (o settingsOutput) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(o) }

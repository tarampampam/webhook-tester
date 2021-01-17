package get

type responseLimits struct {
	MaxRequests        uint16 `json:"max_requests"`
	SessionLifetimeSec uint32 `json:"session_lifetime_sec"`
}

type pusher struct {
	Key     string `json:"key"`
	Cluster string `json:"cluster"`
}

type response struct {
	Version string         `json:"version"`
	Pusher  pusher         `json:"pusher"`
	Limits  responseLimits `json:"limits"`
}

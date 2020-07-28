package get

type responseLimits struct {
	MaxRequests        uint16 `json:"max_requests"`
	SessionLifetimeSec uint32 `json:"session_lifetime_sec"`
}

type response struct {
	Version string         `json:"version"`
	Limits  responseLimits `json:"limits"`
}

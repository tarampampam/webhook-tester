package broadcast

import (
	"net/http"

	"github.com/pusher/pusher-http-go"
)

// PusherOption allows to more tiny pusher setup.
type PusherOption func(*Pusher)

// WithPusherHTTPClient setups allows to pass custom HTTP client implementation.
func WithPusherHTTPClient(httpClient *http.Client) PusherOption {
	return func(p *Pusher) { p.cl.HTTPClient = httpClient }
}

// Pusher is a publisher, that uses 'pusher.com' for events broadcasting.
type Pusher struct {
	cl *pusher.Client
}

// NewPusher creates new pusher publisher.
func NewPusher(appID, key, secret, cluster string, options ...PusherOption) *Pusher {
	p := Pusher{
		cl: &pusher.Client{
			AppID:   appID,
			Key:     key,
			Secret:  secret,
			Cluster: cluster,
			Secure:  true,
		},
	}

	for i := 0; i < len(options); i++ {
		options[i](&p)
	}

	return &p
}

// Publish an event into passed channel.
func (p *Pusher) Publish(channel string, event Event) error {
	return p.cl.Trigger(channel, event.Name(), event.Data())
}

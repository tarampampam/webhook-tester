package pusher

import (
	"github.com/pusher/pusher-http-go"
)

type Broadcaster struct {
	pusher *pusher.Client
}

func NewBroadcaster(appID, key, secret, cluster string) *Broadcaster {
	return &Broadcaster{
		pusher: &pusher.Client{
			AppID:   appID,
			Key:     key,
			Secret:  secret,
			Cluster: cluster,
			Secure:  true,
		},
	}
}

func (b *Broadcaster) Publish(channel, eventName string, data interface{}) error {
	return b.pusher.Trigger(channel, eventName, data)
}

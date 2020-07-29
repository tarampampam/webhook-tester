package broadcast

type Broadcaster interface {
	Publish(channel, eventName string, data interface{}) error
}

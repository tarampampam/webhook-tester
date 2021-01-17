package broadcast

// Broadcaster allows to broadcast events.
type Broadcaster interface {
	// Publish sends event with data in passed channel.
	Publish(channel, eventName string, data interface{}) error
}

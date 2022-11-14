// Package pubsub is used for events publishing and subscribing for them.
package pubsub

// Publisher allows to publish Event*s.
type Publisher interface {
	// Publish an event into passed channel.
	Publish(channelName string, event Event) error
}

// Subscriber allows to Subscribe and Unsubscribe for Event*s.
type Subscriber interface {
	// Subscribe to the named channel and receive Event's into the passed channel.
	//
	// Keep in mind - passed channel (chan) must be created on the caller side and channels without active readers
	// (or closed too early) can block application working (or break it at all).
	//
	// Also do not forget to Unsubscribe from the channel.
	Subscribe(channelName string, channel chan<- Event) error

	// Unsubscribe the subscription to the named channel for the passed events channel.
	Unsubscribe(channelName string, channel chan Event) error
}

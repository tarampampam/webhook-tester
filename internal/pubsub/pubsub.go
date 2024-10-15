package pubsub

import "context"

type (
	Publisher[T any] interface {
		// Publish an event into the channel with the passed name.
		Publish(_ context.Context, channel string, event T) error
	}

	Subscriber[T any] interface {
		// Subscribe to the named channel. The returned channel will receive events.
		// The returned function should be called to unsubscribe.
		Subscribe(_ context.Context, channel string) (_ <-chan T, unsubscribe func(), _ error)
	}
)

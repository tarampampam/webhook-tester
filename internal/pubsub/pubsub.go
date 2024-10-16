package pubsub

import "context"

type (
	Publisher[T any] interface {
		// Publish an event into the topic.
		Publish(_ context.Context, topic string, event T) error
	}

	Subscriber[T any] interface {
		// Subscribe to the topic. The returned channel will receive events.
		// The returned function should be called to unsubscribe.
		Subscribe(_ context.Context, topic string) (_ <-chan T, unsubscribe func(), _ error)
	}
)

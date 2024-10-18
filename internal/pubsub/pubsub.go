package pubsub

import (
	"context"
)

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

type PubSub[T any] interface {
	Publisher[T]
	Subscriber[T]
}

type (
	CapturedRequest struct {
		ID                 string       `json:"id"`
		ClientAddr         string       `json:"client_addr"`
		Method             string       `json:"method"`
		Body               []byte       `json:"body"`
		Headers            []HttpHeader `json:"headers"`
		URL                string       `json:"url"`
		CreatedAtUnixMilli int          `json:"created_at_unix_milli"`
	}

	HttpHeader struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
)

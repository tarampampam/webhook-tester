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
	RequestEvent struct {
		Action  RequestAction `json:"action"`
		Request *Request      `json:"request"`
	}

	Request struct {
		ID                 string       `json:"id"`
		ClientAddr         string       `json:"client_addr"`
		Method             string       `json:"method"`
		Headers            []HttpHeader `json:"headers"`
		URL                string       `json:"url"`
		CreatedAtUnixMilli int64        `json:"created_at_unix_milli"`
	}

	HttpHeader struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	RequestAction = string
)

const (
	RequestActionCreate RequestAction = "create" // create a request
	RequestActionDelete RequestAction = "delete" // delete a request
	RequestActionClear  RequestAction = "clear"  // delete all requests
)

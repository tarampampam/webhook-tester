package pubsub

import (
	"context"
	"errors"
	"time"
)

// ErrClosed is returned by Publish and Subscribe after Close has been called.
var ErrClosed = errors.New("pubsub is closed")

// Publisher publishes events to named topics.
type Publisher interface {
	// Publish an event into the topic.
	Publish(_ context.Context, topic string, event RequestEvent) error
}

// Subscriber subscribes to named topics and receives events.
type Subscriber interface {
	// Subscribe to the topic. The returned channel will receive events.
	// The returned function should be called to unsubscribe.
	Subscribe(_ context.Context, topic string) (_ <-chan RequestEvent, unsubscribe func(), _ error)
}

// PubSub combines the Publisher and Subscriber interfaces.
type PubSub interface {
	Publisher
	Subscriber
}

// RequestAction describes the type of change to a captured request.
type RequestAction string

const (
	RequestActionCreate RequestAction = "create" // create a request
	RequestActionDelete RequestAction = "delete" // delete a request
	RequestActionClear  RequestAction = "clear"  // delete all requests
)

type (
	// RequestEvent is the event published when a captured request is created, deleted, or all requests are removed.
	RequestEvent struct {
		Action  RequestAction
		Request *RequestData
	}

	// RequestData holds the details of a captured HTTP request.
	RequestData struct {
		ID         string
		ClientAddr string
		Method     string
		Headers    []HttpHeader
		URL        string
		CreatedAt  time.Time
	}

	// HttpHeader represents a single HTTP header name-value pair.
	HttpHeader struct {
		Name, Value string
	}
)

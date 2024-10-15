package pubsub

import (
	"context"
	"sync"
)

type InMemory[T any] struct {
	subsMu sync.Mutex
	subs   map[string]map[chan<- T]chan struct{} // map[channel_name]map[subscribed_channel]stop_channel
}

var ( // ensure interface implementation
	_ Publisher[any]  = (*InMemory[any])(nil)
	_ Subscriber[any] = (*InMemory[any])(nil)
)

func NewInMemory[T any]() *InMemory[T] {
	return &InMemory[T]{subs: make(map[string]map[chan<- T]chan struct{})}
}

func (ps *InMemory[T]) Publish(ctx context.Context, channel string, event T) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[channel]; !exists { // if there are no subscribers - do not publish
		return nil
	}

	for target, stop := range ps.subs[channel] {
		go func(target chan<- T, stop <-chan struct{}) {
			select { // first, check if we need to stop to avoid blocking on probably already closed channel
			case <-stop:
			case <-ctx.Done():
			default:
				select { // then, try to send an event
				case <-stop:
				case <-ctx.Done():
				case target <- event:
				}
			}
		}(target, stop)
	}

	return nil
}

func (ps *InMemory[T]) Subscribe(ctx context.Context, channel string) (<-chan T, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, func() { /* noop */ }, err // context is done
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[channel]; !exists { // create a subscription if needed
		ps.subs[channel] = make(map[chan<- T]chan struct{})
	}

	var sub, stop = make(chan T), make(chan struct{})

	ps.subs[channel][sub] = stop

	return sub, sync.OnceFunc(func() {
		close(stop) // notify to stop

		ps.subsMu.Lock()
		defer ps.subsMu.Unlock()

		// remove subscription
		delete(ps.subs[channel], sub)

		// remove channel if there are no subscribers
		if len(ps.subs[channel]) == 0 {
			delete(ps.subs, channel)
		}

		close(sub) // close channel
	}), nil
}

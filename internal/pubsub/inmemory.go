package pubsub

import (
	"context"
	"sync"
)

type InMemory[T any] struct {
	subsMu sync.Mutex
	subs   map[string]map[chan<- T]chan struct{} // map[topic]map[subscription]stop
}

var ( // ensure interface implementation
	_ Publisher[any]  = (*InMemory[any])(nil)
	_ Subscriber[any] = (*InMemory[any])(nil)
)

func NewInMemory[T any]() *InMemory[T] {
	return &InMemory[T]{subs: make(map[string]map[chan<- T]chan struct{})}
}

func (ps *InMemory[T]) Publish(ctx context.Context, topic string, event T) error {
	if err := ctx.Err(); err != nil {
		return err // context is done
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[topic]; !exists { // if there are no subscribers - do not publish
		return nil
	}

	for sub, stop := range ps.subs[topic] {
		go func(sub chan<- T, stop <-chan struct{}) {
			select { // first, check if we need to stop to avoid blocking on probably already closed channel
			case <-stop:
			case <-ctx.Done():
			default:
				select { // then, try to send an event
				case <-stop:
				case <-ctx.Done():
				case sub <- event:
				}
			}
		}(sub, stop)
	}

	return nil
}

func (ps *InMemory[T]) Subscribe(ctx context.Context, topic string) (<-chan T, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, func() { /* noop */ }, err // context is done
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[topic]; !exists { // create a subscription if needed
		ps.subs[topic] = make(map[chan<- T]chan struct{})
	}

	var sub, stop = make(chan T, 1), make(chan struct{})

	ps.subs[topic][sub] = stop

	return sub, sync.OnceFunc(func() {
		close(stop) // notify to stop

		ps.subsMu.Lock()

		// remove subscription
		delete(ps.subs[topic], sub)

		// remove channel if there are no subscribers
		if len(ps.subs[topic]) == 0 {
			delete(ps.subs, topic)
		}

		ps.subsMu.Unlock()

		// empty the sub channel
		for len(sub) > 0 {
			<-sub
		}

		close(sub) // close channel
	}), nil
}

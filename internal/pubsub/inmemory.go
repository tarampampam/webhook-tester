package pubsub

import (
	"context"
	"sync"
)

type (
	InMemory[T any] struct {
		subsMu sync.Mutex
		subs   map[ /* topic */ string]map[ /* subscription */ chan<- T]*inMemorySubState
	}

	inMemorySubState struct {
		wg   sync.WaitGroup
		stop chan struct{}
	}
)

var ( // ensure interface implementation
	_ Publisher[any]  = (*InMemory[any])(nil)
	_ Subscriber[any] = (*InMemory[any])(nil)
)

func NewInMemory[T any]() *InMemory[T] {
	return &InMemory[T]{subs: make(map[string]map[chan<- T]*inMemorySubState)}
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

	for sub, state := range ps.subs[topic] {
		state.wg.Add(1) // tell the subscriber that we are about to send an event

		go func(sub chan<- T, stop <-chan struct{}, wg *sync.WaitGroup) {
			defer wg.Done() // notify the subscriber that we are done

			select {
			case <-ctx.Done(): // check the context
			case <-stop: // stopping notification
			case sub <- event: // and in the same time try to send the event
			}
		}(sub, state.stop, &state.wg)
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
		ps.subs[topic] = make(map[chan<- T]*inMemorySubState)
	}

	var sub, state = make(chan T), &inMemorySubState{stop: make(chan struct{})}

	ps.subs[topic][sub] = state

	return sub, sync.OnceFunc(func() {
		close(state.stop) // notify all the publishers to stop

		ps.subsMu.Lock()

		delete(ps.subs[topic], sub) // remove subscription

		if len(ps.subs[topic]) == 0 { // remove channel if there are no subscribers (cleanup)
			delete(ps.subs, topic)
		}

		ps.subsMu.Unlock()

		for len(sub) > 0 {
			<-sub
		}

		state.wg.Wait() // wait until all the publishers are done

		close(sub) // and close the subscription channel
	}), nil
}

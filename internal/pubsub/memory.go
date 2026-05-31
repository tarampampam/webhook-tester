package pubsub

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

// Memory is an in-process pub/sub backed by Go channels.
type Memory struct {
	subsMu sync.RWMutex // guards subs
	subs   map[string]map[chan RequestEvent]*memorySubState

	closed atomic.Bool
}

var ( // compile-time interface assertions
	_ PubSub    = (*Memory)(nil)
	_ io.Closer = (*Memory)(nil)
)

// NewMemory creates a new in-memory pub/sub.
func NewMemory() *Memory {
	return &Memory{subs: make(map[string]map[chan RequestEvent]*memorySubState)}
}

// Publish implements [Publisher].
func (ps *Memory) Publish(ctx context.Context, topic string, event RequestEvent) error {
	// check for the context cancellation and closed state before acquiring the lock
	if err, closed := ctx.Err(), ps.closed.Load(); err != nil || closed {
		if closed {
			err = ErrClosed
		}

		return err
	}

	// snapshot subscribers under the lock - wg.Add must happen here to prevent a race with the concurrent
	// unsubscribes wg.Wait
	ps.subsMu.RLock()

	type topicSnapshot struct {
		ch   chan<- RequestEvent
		stop <-chan struct{}
		wg   *sync.WaitGroup
	}

	subs := make([]topicSnapshot, 0, len(ps.subs[topic]))
	for ch, state := range ps.subs[topic] {
		state.wg.Add(1)
		subs = append(subs, topicSnapshot{ch, state.stop, &state.wg})
	}

	ps.subsMu.RUnlock()

	// send the event to all subscribers in parallel; if a subscriber is too slow or blocked, it won't affect the others
	for _, s := range subs {
		go func(s topicSnapshot) {
			defer s.wg.Done()

			select {
			case <-ctx.Done():
			case <-s.stop:
			case s.ch <- event:
			}
		}(s)
	}

	return nil
}

// Subscribe implements [Subscriber].
func (ps *Memory) Subscribe(ctx context.Context, topic string) (<-chan RequestEvent, func(), error) {
	// check for the context cancellation and closed state before acquiring the lock
	if err, closed := ctx.Err(), ps.closed.Load(); err != nil || closed {
		if closed {
			err = ErrClosed
		}

		return nil, func() {}, err
	}

	ps.subsMu.Lock()

	// check closed state again exactly after acquiring the lock - to prevent a race with a concurrent Close
	if ps.closed.Load() {
		ps.subsMu.Unlock()

		return nil, func() {}, ErrClosed
	}

	// allocate the map for this topic if it doesn't exist
	if _, exists := ps.subs[topic]; !exists {
		ps.subs[topic] = make(map[chan RequestEvent]*memorySubState)
	}

	var (
		subCh = make(chan RequestEvent)
		state = memorySubState{stop: make(chan struct{})}
	)

	unsub := sync.OnceFunc(func() {
		// remove from the map first so no new Publish goroutines are spawned for this subscriber
		ps.subsMu.Lock()
		delete(ps.subs[topic], subCh)

		if len(ps.subs[topic]) == 0 {
			delete(ps.subs, topic)
		}
		ps.subsMu.Unlock()

		close(state.stop) // abort in-flight publish goroutines for this subscriber
		state.wg.Wait()   // wait until all of them have exited
		close(subCh)      // safe to close: no more senders
	})

	state.unsub = unsub
	ps.subs[topic][subCh] = &state
	ps.subsMu.Unlock()

	// auto-unsubscribe when the caller's context is canceled
	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				unsub()
			case <-state.stop: // already unsubscribed manually; exit to avoid goroutine leak
			}
		}()
	}

	return subCh, unsub, nil
}

// Close implements [io.Closer]. It unsubscribes all active subscribers, releases their channels and goroutines,
// and marks the instance as closed. Any subsequent call to Publish or Subscribe returns [ErrClosed].
func (ps *Memory) Close() error {
	if !ps.closed.CompareAndSwap(false, true) {
		return ErrClosed
	}

	// snapshot all unsub functions and clear the map before releasing the lock; unsub must be called outside
	// the lock because it re-acquires ps.mu internally
	ps.subsMu.Lock()

	unsubs := make([]func(), 0, len(ps.subs))
	for _, topicSubs := range ps.subs {
		for _, state := range topicSubs {
			unsubs = append(unsubs, state.unsub)
		}
	}

	ps.subs = make(map[string]map[chan RequestEvent]*memorySubState)
	ps.subsMu.Unlock()

	// call all unsub functions to clean up subscribers
	for _, unsub := range unsubs {
		unsub()
	}

	return nil
}

type memorySubState struct {
	wg   sync.WaitGroup
	stop chan struct{}

	unsub func() // stored so Close can call it without replicating teardown logic
}

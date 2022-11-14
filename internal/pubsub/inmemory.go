package pubsub

import (
	"errors"
	"sync"
)

// InMemory publisher/subscriber uses memory for events publishing and delivering to the subscribers. Useful for
// application "single node" mode running or unit testing.
//
// Publishing/subscribing events order is NOT guaranteed.
//
// Node: Do not forget to Close it after all. Closed publisher/subscriber cannot be opened back.
type InMemory struct {
	subsMu sync.Mutex
	subs   map[string]map[chan<- Event]chan struct{} // map<channel name>map<subscribed ch.>stopping signal ch.

	closedMu sync.Mutex
	closed   bool
}

// NewInMemory creates new inmemory publisher/subscriber.
func NewInMemory() *InMemory {
	return &InMemory{
		subs: make(map[string]map[chan<- Event]chan struct{}),
	}
}

func (ps *InMemory) createSubscriptionIfNeeded(channelName string) {
	ps.subsMu.Lock()
	if _, exists := ps.subs[channelName]; !exists {
		ps.subs[channelName] = make(map[chan<- Event]chan struct{})
	}
	ps.subsMu.Unlock()
}

// Publish an event into passed channel. Publishing is non-blocking operation.
func (ps *InMemory) Publish(channelName string, event Event) error {
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	ps.createSubscriptionIfNeeded(channelName)

	ps.subsMu.Lock()

	for target, stop := range ps.subs[channelName] {
		go func(target chan<- Event, stop <-chan struct{}) { // send an event without blocking
			select {
			case <-stop:
				return

			case target <- event: // <- panic can be occurred here (if channel was closed too early outside)
			}
		}(target, stop)
	}

	ps.subsMu.Unlock()

	return nil
}

// Subscribe to the named channel and receive Event's into the passed channel. Channel must be created on the calling
// side and NOT to be closed until subscription is not Unsubscribe*ed.
//
// Note: do not forget to call Unsubscribe when all is done.
func (ps *InMemory) Subscribe(channelName string, channel chan<- Event) error {
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	ps.createSubscriptionIfNeeded(channelName)

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[channelName][channel]; exists {
		return errors.New("already subscribed")
	}

	ps.subs[channelName][channel] = make(chan struct{}, 1)

	return nil
}

// Unsubscribe the subscription to the named channel for the passed events channel. Be careful with channel closing,
// this can call the panics if some Event's scheduled for publishing.
func (ps *InMemory) Unsubscribe(channelName string, channel chan Event) error {
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[channelName]; !exists {
		return errors.New("subscription does not exists")
	}

	if _, exists := ps.subs[channelName][channel]; !exists {
		return errors.New("channel was not subscribed")
	}

	// send "cancellation" signal to all publishing goroutines
	ps.subs[channelName][channel] <- struct{}{}
	close(ps.subs[channelName][channel])

	// unsubscribe channel
	delete(ps.subs[channelName], channel)

	// cleanup subscriptions map, if needed
	if len(ps.subs[channelName]) == 0 {
		delete(ps.subs, channelName)
	}

	return nil
}

func (ps *InMemory) isClosed() (isClosed bool) {
	ps.closedMu.Lock()
	isClosed = ps.closed
	ps.closedMu.Unlock()

	return
}

// Close this publisher/subscriber. This function can be called only once.
func (ps *InMemory) Close() error {
	if ps.isClosed() {
		return errors.New("already closed")
	}

	ps.closedMu.Lock()
	ps.closed = true
	ps.closedMu.Unlock()

	ps.subsMu.Lock()
	for channelName, channels := range ps.subs {
		for _, cancelCh := range channels {
			// send "cancellation" signal to the all publishing goroutines
			cancelCh <- struct{}{}
			close(cancelCh)
		}

		delete(ps.subs, channelName)
	}
	ps.subsMu.Unlock()

	return nil
}

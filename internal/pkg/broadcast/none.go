package broadcast

import "sync"

// None broadcaster is useful for preventing event publishing "outside" of the application. It can be used as a plug for
// the "real" code or for the unit tests.
type None struct {
	mu          sync.Mutex
	err         error
	lastChannel string
	lastEvent   Event
}

// Publish an event into passed channel.
func (n *None) Publish(channel string, event Event) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.lastChannel, n.lastEvent = channel, event

	if n.err != nil {
		return n.err
	}

	return nil
}

// SetError allows to set some error, that will be returned on Publish method calling. Pass <nil> to unset.
func (n *None) SetError(err error) {
	n.mu.Lock()
	n.err = err
	n.mu.Unlock()
}

// LastPublishedEvent returns last published channel name and event.
func (n *None) LastPublishedEvent() (string, Event) {
	n.mu.Lock()
	ch, e := n.lastChannel, n.lastEvent
	n.mu.Unlock()

	return ch, e
}

package broadcast

import "sync"

// None broadcaster is useful for preventing event publishing "outside" of the application. It can be used as a plug
// for the "real" code or for the unit tests.
type None struct {
	mu sync.Mutex
	l  []func(ch string, e Event)
}

// Publish an event into passed channel.
func (n *None) Publish(channel string, event Event) error {
	n.mu.Lock()
	if len(n.l) > 0 {
		for i := 0; i < len(n.l); i++ {
			n.l[i](channel, event)
		}
	}
	n.mu.Unlock()

	return nil
}

// OnPublish allows to attack your handler on Publish function calling.
func (n *None) OnPublish(f func(ch string, e Event)) {
	n.mu.Lock()
	if n.l == nil {
		n.l = make([]func(string, Event), 0, 1) // "lazy" init
	}

	n.l = append(n.l, f)
	n.mu.Unlock()
}

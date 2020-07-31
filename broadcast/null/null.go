// Fake broadcaster, just for a test

package null

import "sync"

type Broadcaster struct {
	Error error

	m sync.Mutex
	lastPublishedChannel,
	lastPublishedEventName string
	lastPublishedData interface{}
}

func (b *Broadcaster) Publish(channel, eventName string, data interface{}) error {
	b.m.Lock()
	defer b.m.Unlock()

	b.lastPublishedChannel = channel
	b.lastPublishedEventName = eventName
	b.lastPublishedData = data

	return b.Error
}

func (b *Broadcaster) GetLastPublishedChannel() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.lastPublishedChannel
}

func (b *Broadcaster) GetLastPublishedEventName() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.lastPublishedEventName
}

func (b *Broadcaster) GetLastPublishedData() interface{} {
	b.m.Lock()
	defer b.m.Unlock()

	return b.lastPublishedData
}

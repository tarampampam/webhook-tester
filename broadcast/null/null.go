// Fake broadcaster, just for a test

package null

type Broadcaster struct {
	Error error
}

func (b *Broadcaster) Publish(_, _ string, _ interface{}) error {
	return b.Error
}

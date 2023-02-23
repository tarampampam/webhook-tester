package pubsub_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/internal/pubsub"
)

func TestInMemory_PublishErrors(t *testing.T) {
	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	assert.EqualError(t, ps.Publish("", pubsub.NewRequestRegisteredEvent("bar")), "empty channel name is not allowed")
}

func TestInMemory_PublishAndReceive(t *testing.T) {
	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var event1, event2 = pubsub.NewRequestRegisteredEvent("bar"), pubsub.NewRequestRegisteredEvent("baz")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() { // each of subscriber must receive a copy of published event
			defer wg.Done()

			ch := make(chan pubsub.Event)
			defer close(ch)

			assert.NoError(t, ps.Subscribe("foo", ch))

			defer func() { assert.NoError(t, ps.Unsubscribe("foo", ch)) }()

			receivedEvents := make([]pubsub.Event, 0, 2)

			for j := 0; j < cap(receivedEvents); j++ {
				select {
				case <-ctx.Done():
					t.Error(ctx.Err())

					return

				case e := <-ch:
					receivedEvents = append(receivedEvents, e)
				}
			}

			assert.Len(t, receivedEvents, 2)

			for j := 0; j < len(receivedEvents); j++ {
				if event := receivedEvents[j]; event != event1 && event != event2 {
					t.Error("received events must be one of expected")
				}
			}
		}()
	}

	runtime.Gosched()
	<-time.After(time.Millisecond) // make sure that all subscribes was subscribed successfully

	assert.NoError(t, ps.Publish("foo", event1))
	assert.NoError(t, ps.Publish("foo", event2))

	wg.Wait()

	assert.NoError(t, ps.Close())
}

func TestInMemory_Close(t *testing.T) {
	ps := pubsub.NewInMemory()

	assert.NoError(t, ps.Close())

	ch := make(chan pubsub.Event)

	assert.EqualError(t, ps.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")), "closed")
	assert.EqualError(t, ps.Subscribe("foo", ch), "closed")
	assert.EqualError(t, ps.Unsubscribe("foo", ch), "closed")
	assert.EqualError(t, ps.Close(), "already closed")
}

func TestInMemory_Unsubscribe(t *testing.T) {
	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	ch1, ch2 := make(chan pubsub.Event, 3), make(chan pubsub.Event, 3)
	// defer func() { close(ch1); close(ch2) }() // <- do not do thue due race reasons

	assert.NoError(t, ps.Subscribe("foo", ch1))
	assert.NoError(t, ps.Subscribe("foo", ch2))

	assert.EqualError(t, ps.Unsubscribe("", ch1), "empty channel name is not allowed")

	assert.NoError(t, ps.Unsubscribe("foo", ch2))
	assert.EqualError(t, ps.Unsubscribe("foo", ch2), "channel was not subscribed") // repeated op
	assert.EqualError(t, ps.Unsubscribe("baz", ch2), "subscription does not exists")

	assert.NoError(t, ps.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")))

	runtime.Gosched()
	<-time.After(time.Millisecond)

	assert.Len(t, ch1, 1)
	assert.Len(t, ch2, 0)
}

func TestInMemory_Subscribe(t *testing.T) {
	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	ch := make(chan pubsub.Event)
	defer close(ch)

	assert.NoError(t, ps.Subscribe("foo", ch))

	defer func() { assert.NoError(t, ps.Unsubscribe("foo", ch)) }()

	assert.EqualError(t, ps.Subscribe("", ch), "empty channel name is not allowed")

	assert.EqualError(t, ps.Subscribe("foo", ch), "already subscribed") // repeated
}

func TestInMemory_UnsubscribeWithChannelClosingWithoutReading(t *testing.T) {
	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	for i := 0; i < 1_000; i++ {
		ch := make(chan pubsub.Event)

		assert.NoError(t, ps.Subscribe("foo", ch))

		assert.NoError(t, ps.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")))

		assert.NoError(t, ps.Unsubscribe("foo", ch))
	}

	for i := 0; i < 1_000; i++ {
		ps2 := pubsub.NewInMemory()
		ch := make(chan pubsub.Event)

		assert.NoError(t, ps2.Subscribe("foo", ch))

		assert.NoError(t, ps2.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")))

		assert.NoError(t, ps2.Close())
	}
}

func BenchmarkInMemory_PublishAndReceive(b *testing.B) {
	b.ReportAllocs()

	ps := pubsub.NewInMemory()
	defer func() { _ = ps.Close() }()

	ch := make(chan pubsub.Event)
	defer close(ch)

	if err := ps.Subscribe("foo", ch); err != nil {
		b.Error(err)
	}

	defer func() { _ = ps.Unsubscribe("foo", ch) }()

	event := pubsub.NewRequestRegisteredEvent("bar")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := ps.Publish("foo", event); err != nil {
			b.Error(err)
		}

		if e := <-ch; e != event {
			b.Error("wrong event received")
		}
	}
}

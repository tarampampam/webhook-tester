package pubsub_test

import (
	"bytes"
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
)

func TestRedis_PublishErrors(t *testing.T) {
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	assert.EqualError(t, ps.Publish("", pubsub.NewRequestRegisteredEvent("bar")), "empty channel name is not allowed")
}

func TestRedis_PublishAndReceive(t *testing.T) {
	t.Parallel()

	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var event1, event2 = pubsub.NewRequestRegisteredEvent("bar"), pubsub.NewRequestRegisteredEvent("baz")

	eventsAreEquals := func(t *testing.T, a, b pubsub.Event) bool {
		t.Helper()

		if !bytes.Equal(a.Data(), b.Data()) {
			return false
		}

		if a.Name() != b.Name() {
			return false
		}

		return true
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
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
				if e := receivedEvents[j]; !eventsAreEquals(t, e, event1) && !eventsAreEquals(t, e, event2) {
					t.Errorf("received events must be one of expected, but got: %+v", e)
				}
			}
		}()
	}

	<-time.After(time.Millisecond) // make sure that all subscribes was subscribed successfully

	assert.NoError(t, ps.Publish("foo", event1))
	assert.NoError(t, ps.Publish("foo", event2))

	wg.Wait()
}

func TestRedis_Close(t *testing.T) {
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	assert.NoError(t, ps.Close())

	ch := make(chan pubsub.Event)

	assert.EqualError(t, ps.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")), "closed")
	assert.EqualError(t, ps.Subscribe("foo", ch), "closed")
	assert.EqualError(t, ps.Unsubscribe("foo", ch), "closed")
	assert.EqualError(t, ps.Close(), "already closed")
}

func TestRedis_Unsubscribe(t *testing.T) {
	t.Parallel()

	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	for i := 0; i < 20; i++ {
		t.Run("attempt #"+strconv.Itoa(i), func(t *testing.T) {
			ch1, ch2 := make(chan pubsub.Event, 1), make(chan pubsub.Event, 1)

			assert.NoError(t, ps.Subscribe("foo", ch1))

			<-time.After(time.Millisecond * 5)

			assert.NoError(t, ps.Subscribe("foo", ch2)) // will be not unsubscribed for a test
			assert.EqualError(t, ps.Unsubscribe("", ch1), "empty channel name is not allowed")

			assert.NoError(t, ps.Unsubscribe("foo", ch1))
			assert.EqualError(t, ps.Unsubscribe("baz", ch1), "subscription does not exists")
			assert.EqualError(t, ps.Unsubscribe("foo", ch1), "channel was not subscribed") // repeated op

			assert.NoError(t, ps.Publish("foo", pubsub.NewRequestRegisteredEvent("bar")))

			<-time.After(time.Millisecond * 15)

			assert.Len(t, ch1, 0)
			assert.Len(t, ch2, 1)

			// close(ch1); close(ch2) // <- do not do thue due race reasons

			assert.NoError(t, ps.Unsubscribe("foo", ch2))
		})
	}
}

func TestRedis_Subscribe(t *testing.T) {
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	ch := make(chan pubsub.Event)
	defer close(ch)

	assert.NoError(t, ps.Subscribe("foo", ch))

	assert.EqualError(t, ps.Subscribe("", ch), "empty channel name is not allowed")

	assert.EqualError(t, ps.Subscribe("foo", ch), "already subscribed") // repeated
}

func TestRedis_UnsubscribeWithChannelClosingWithoutReading(t *testing.T) {
	t.Parallel()

	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
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

func BenchmarkRedis_PublishAndReceive(b *testing.B) {
	b.ReportAllocs()

	mini, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mini.Close()

	ps := pubsub.NewRedis(context.Background(), redis.NewClient(&redis.Options{Addr: mini.Addr()}))
	defer func() { _ = ps.Close() }()

	ch := make(chan pubsub.Event)
	defer close(ch)

	if err = ps.Subscribe("foo", ch); err != nil {
		b.Fatal(err)
	}

	defer func() { _ = ps.Unsubscribe("foo", ch) }()

	event := pubsub.NewRequestRegisteredEvent("bar")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err = ps.Publish("foo", event); err != nil {
			b.Fatal(err)
		}

		if e := <-ch; !bytes.Equal(e.Data(), event.Data()) || e.Name() != event.Name() {
			b.Fatal("wrong event received")
		}
	}
}

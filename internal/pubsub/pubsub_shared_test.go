package pubsub_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// PubSubFactory creates a fresh, isolated PubSub instance for a single test or benchmark case.
// The factory must register tb.Cleanup to release any resources when the test or benchmark ends.
type PubSubFactory func(tb testing.TB) pubsub.PubSub

// RunSuite exercises the full [pubsub.PubSub] contract.
// Call it from each implementation's test file with an appropriate factory.
func RunSuite(t *testing.T, factory PubSubFactory) {
	t.Helper()

	t.Run("Subscribe", func(t *testing.T) { t.Parallel(); testSubscribe(t, factory) })
	t.Run("Publish", func(t *testing.T) { t.Parallel(); testPublish(t, factory) })
	t.Run("ContextAutoUnsubscribe", func(t *testing.T) { t.Parallel(); testContextAutoUnsubscribe(t, factory) })
	t.Run("RaceProvocation", func(t *testing.T) { t.Parallel(); testRaceProvocation(t, factory) })
	t.Run("Close", func(t *testing.T) { t.Parallel(); testClose(t, factory) })
}

func testSubscribe(t *testing.T, factory PubSubFactory) {
	t.Helper()

	t.Run("returns non-nil channel and unsubscribe func", func(t *testing.T) {
		t.Parallel()

		ch, unsub, err := factory(t).Subscribe(t.Context(), "topic")
		assert.NoError(t, err)
		assert.NotNil(t, ch)
		assert.NotNil(t, unsub)

		unsub()
	})

	t.Run("already-canceled context returns error and non-blocking channel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		ch, unsub, err := factory(t).Subscribe(ctx, "topic")
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, ch)
		assert.NotNil(t, unsub)

		unsub()
	})

	t.Run("unsubscribe closes the channel", func(t *testing.T) {
		t.Parallel()

		ch, unsub, err := factory(t).Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		unsub()

		select {
		case _, ok := <-ch:
			assert.False(t, ok)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for channel to close after unsubscribe")
		}
	})

	t.Run("unsubscribe is idempotent", func(t *testing.T) {
		t.Parallel()

		_, unsub, err := factory(t).Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		assert.NotPanics(t, func() {
			unsub()
			unsub()
			unsub()
		})
	})
}

func testPublish(t *testing.T, factory PubSubFactory) {
	t.Helper()

	t.Run("returns nil for topic with no subscribers", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, factory(t).Publish(t.Context(), "empty-topic", pubsub.RequestEvent{
			Action: pubsub.RequestActionCreate,
		}))
	})

	t.Run("already-canceled context returns ctx.Err()", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		assert.ErrorIs(t, factory(t).Publish(ctx, "topic", pubsub.RequestEvent{}), context.Canceled)
	})

	t.Run("subscriber receives the exact event", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		ch, unsub, err := ps.Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		defer unsub()

		want := pubsub.RequestEvent{
			Action: pubsub.RequestActionCreate,
			Request: &pubsub.RequestData{
				ID:         "r1",
				ClientAddr: "1.2.3.4",
				Method:     "POST",
				URL:        "https://example.com/webhook",
			},
		}

		assert.NoError(t, ps.Publish(t.Context(), "topic", want))

		select {
		case got := <-ch:
			assert.DeepEqual(t, want, got)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("all RequestEvent fields survive the round-trip", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		ch, unsub, err := ps.Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		defer unsub()

		want := pubsub.RequestEvent{
			Action: pubsub.RequestActionCreate,
			Request: &pubsub.RequestData{
				ID:         "req-id-abc123",
				ClientAddr: "192.168.1.100:54321",
				Method:     "POST",
				Headers: []pubsub.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom-Header", Value: "custom-value"},
				},
				URL:       "https://example.com/webhook?foo=bar&baz=qux",
				CreatedAt: time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC),
			},
		}

		assert.NoError(t, ps.Publish(t.Context(), "topic", want))

		select {
		case got := <-ch:
			assert.DeepEqual(t, want, got)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("all subscribers on the same topic receive the event", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		const n = 5

		channels := make([]<-chan pubsub.RequestEvent, n)
		unsubs := make([]func(), n)

		for i := range n {
			ch, unsub, err := ps.Subscribe(t.Context(), "topic")
			assert.NoError(t, err)

			channels[i] = ch
			unsubs[i] = unsub
		}

		defer func() {
			for _, u := range unsubs {
				u()
			}
		}()

		want := pubsub.RequestEvent{Action: pubsub.RequestActionDelete}

		assert.NoError(t, ps.Publish(t.Context(), "topic", want))

		for i, ch := range channels {
			select {
			case got := <-ch:
				assert.DeepEqual(t, want, got)
			case <-time.After(time.Second):
				t.Fatalf("subscriber %d: timed out waiting for event", i)
			}
		}
	})

	t.Run("subscribers on different topics are isolated", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		chA, unsubA, err := ps.Subscribe(t.Context(), "topic-a")
		assert.NoError(t, err)

		defer unsubA()

		chB, unsubB, err := ps.Subscribe(t.Context(), "topic-b")
		assert.NoError(t, err)

		defer unsubB()

		assert.NoError(t, ps.Publish(t.Context(), "topic-a", pubsub.RequestEvent{Action: pubsub.RequestActionCreate}))

		select {
		case <-chA:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event on topic-a")
		}

		select {
		case got := <-chB:
			t.Fatalf("unexpected event received on topic-b: %v", got)
		case <-time.After(20 * time.Millisecond):
			// expected: topic-b subscriber must not receive events from topic-a
		}
	})

	t.Run("publish after unsubscribe is a no-op for that subscriber", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		ch, unsub, err := ps.Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		unsub()

		for range ch {
		} // drain until closed

		// publishing to the now-empty topic must not block or panic
		assert.NoError(t, ps.Publish(t.Context(), "topic", pubsub.RequestEvent{Action: pubsub.RequestActionClear}))
	})

	t.Run("all action types are delivered unchanged", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		ch, unsub, err := ps.Subscribe(t.Context(), "topic")
		assert.NoError(t, err)

		defer unsub()

		for _, action := range []pubsub.RequestAction{
			pubsub.RequestActionCreate,
			pubsub.RequestActionDelete,
			pubsub.RequestActionClear,
		} {
			assert.NoError(t, ps.Publish(t.Context(), "topic", pubsub.RequestEvent{Action: action}))

			select {
			case got := <-ch:
				assert.Equal(t, action, got.Action)
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for action %q", action)
			}
		}
	})
}

func testContextAutoUnsubscribe(t *testing.T, factory PubSubFactory) {
	t.Helper()

	t.Run("context cancellation closes the channel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())

		ch, _, err := factory(t).Subscribe(ctx, "topic")
		assert.NoError(t, err)

		cancel()

		select {
		case _, ok := <-ch:
			assert.False(t, ok) // channel must be closed
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for channel to close after context cancellation")
		}
	})

	t.Run("manual unsubscribe before context cancel does not panic", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		_, unsub, err := factory(t).Subscribe(ctx, "topic")
		assert.NoError(t, err)

		assert.NotPanics(t, func() {
			unsub()
			cancel()
		})
	})

	t.Run("context cancellation with in-flight publish does not panic or deadlock", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		ctx, cancel := context.WithCancel(t.Context())

		_, _, err := ps.Subscribe(ctx, "topic") // never read from channel - simulate slow consumer
		assert.NoError(t, err)

		// publish without reading; the goroutine spawned by Publish will block on the channel send
		assert.NoError(t, ps.Publish(t.Context(), "topic", pubsub.RequestEvent{Action: pubsub.RequestActionCreate}))

		// cancel the subscriber context - auto-unsubscribe must drain the in-flight goroutine
		cancel()

		// publish while the subscriber is being torn down must not deadlock or panic
		assert.NoError(t, ps.Publish(t.Context(), "topic", pubsub.RequestEvent{Action: pubsub.RequestActionCreate}))
	})
}

func testRaceProvocation(t *testing.T, factory PubSubFactory) {
	t.Helper()

	t.Run("concurrent publish subscribe unsubscribe", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		const (
			topics     = 5
			goroutines = 20
		)

		var wg sync.WaitGroup

		for i := range goroutines {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				topic := fmt.Sprintf("topic-%d", i%topics)

				ch, unsub, err := ps.Subscribe(t.Context(), topic)
				if err != nil {
					return
				}

				_ = ps.Publish(t.Context(), topic, pubsub.RequestEvent{Action: pubsub.RequestActionCreate})

				select {
				case <-ch:
				default:
				}

				unsub()

				for range ch {
				} // drain until closed
			}(i)
		}

		wg.Wait()
	})

	t.Run("publish to slow subscriber drains when publish context is canceled", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)

		_, unsub, err := ps.Subscribe(t.Context(), "topic") // never read - simulate slow consumer
		assert.NoError(t, err)

		defer unsub()

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		// must not block beyond the context deadline regardless of subscriber readiness
		_ = ps.Publish(ctx, "topic", pubsub.RequestEvent{Action: pubsub.RequestActionCreate})
	})
}

func testClose(t *testing.T, factory PubSubFactory) {
	t.Helper()

	// probe once to decide whether to skip the whole suite
	probe := factory(t)
	if _, ok := probe.(io.Closer); !ok {
		t.Skipf("%T does not implement io.Closer", probe)

		return
	}

	t.Run("active subscribers are closed on Close", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)
		closer := ps.(io.Closer) //nolint:forcetypeassert // guarded by the parent skip above

		ch1, _, err := ps.Subscribe(t.Context(), "topic-a")
		assert.NoError(t, err)

		ch2, _, err := ps.Subscribe(t.Context(), "topic-b")
		assert.NoError(t, err)

		assert.NoError(t, closer.Close())

		for i, ch := range []<-chan pubsub.RequestEvent{ch1, ch2} {
			select {
			case _, ok := <-ch:
				assert.False(t, ok)
			case <-time.After(time.Second):
				t.Fatalf("subscriber %d: timed out waiting for channel to close after Close()", i)
			}
		}
	})

	t.Run("second Close returns ErrClosed", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)
		closer := ps.(io.Closer) //nolint:forcetypeassert // guarded by the parent skip above

		assert.NoError(t, closer.Close())
		assert.ErrorIs(t, closer.Close(), pubsub.ErrClosed)
	})

	t.Run("operations after Close return ErrClosed", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)
		closer := ps.(io.Closer) //nolint:forcetypeassert // guarded by the parent skip above

		assert.NoError(t, closer.Close())

		for name, fn := range map[string]func() error{
			"Publish":   func() error { return ps.Publish(t.Context(), "topic", pubsub.RequestEvent{}) },
			"Subscribe": func() error { _, unsub, err := ps.Subscribe(t.Context(), "topic"); unsub(); return err },
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				assert.ErrorIs(t, fn(), pubsub.ErrClosed)
			})
		}
	})

	t.Run("Subscribe after Close returns a closed channel", func(t *testing.T) {
		t.Parallel()

		ps := factory(t)
		closer := ps.(io.Closer) //nolint:forcetypeassert // guarded by the parent skip above

		assert.NoError(t, closer.Close())

		ch, unsub, err := ps.Subscribe(t.Context(), "topic")
		assert.ErrorIs(t, err, pubsub.ErrClosed)
		assert.Nil(t, ch)
		assert.NotNil(t, unsub)

		unsub()
	})
}

// --------------------------------------------------------------------------------------------------------------------

// RunBenchmarks covers the core [pubsub.PubSub] operations.
// Call it from each implementation's benchmark function with an appropriate factory.
func RunBenchmarks(b *testing.B, factory PubSubFactory) {
	b.Helper()
	b.ReportAllocs()

	b.Run("publish-one-subscriber", func(b *testing.B) {
		ps := factory(b)
		ctx := b.Context()

		ch, unsub, _ := ps.Subscribe(ctx, "topic")
		defer unsub()

		go func() {
			for range ch {
			}
		}() // drain concurrently so publish goroutines can complete

		evt := pubsub.RequestEvent{Action: pubsub.RequestActionCreate, Request: &pubsub.RequestData{ID: "r1"}}

		b.ResetTimer()

		for range b.N {
			_ = ps.Publish(ctx, "topic", evt)
		}
	})

	b.Run("publish-fanout-10-subscribers", func(b *testing.B) {
		ps := factory(b)
		ctx := b.Context()

		const n = 10

		for range n {
			ch, unsub, _ := ps.Subscribe(ctx, "topic")
			defer unsub()

			go func() {
				for range ch {
				}
			}()
		}

		evt := pubsub.RequestEvent{Action: pubsub.RequestActionCreate}

		b.ResetTimer()

		for range b.N {
			_ = ps.Publish(ctx, "topic", evt)
		}
	})

	b.Run("lifecycle", func(b *testing.B) {
		ps := factory(b)
		ctx := b.Context()

		evt := pubsub.RequestEvent{Action: pubsub.RequestActionCreate, Request: &pubsub.RequestData{ID: "r1"}}

		b.ResetTimer()

		for range b.N {
			ch, unsub, _ := ps.Subscribe(ctx, "topic")
			_ = ps.Publish(ctx, "topic", evt)

			<-ch    // receive the single delivered event
			unsub() // blocks until in-flight goroutines exit, then closes ch
		}
	})
}

package pubsub_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
)

type pubSub[T any] interface {
	pubsub.Publisher[T]
	pubsub.Subscriber[T]
}

func testPublishAndReceive(t *testing.T, new func() pubSub[any]) {
	t.Helper()

	const (
		channel1name, channel2name = "foo", "bar"
		event1data, event2data     = "event1", "event2"
	)

	var (
		ps  = new()
		ctx = context.Background()
	)

	var (
		sub1, close1, sub1err = ps.Subscribe(ctx, channel1name)
		sub2, close2, sub2err = ps.Subscribe(ctx, channel2name)
	)

	require.NotNil(t, sub1)
	require.NotNil(t, close1)
	require.NoError(t, sub1err)

	require.NotNil(t, sub2)
	require.NotNil(t, close2)
	require.NoError(t, sub2err)

	t.Run("publish", func(t *testing.T) {
		require.NoError(t, ps.Publish(ctx, channel1name, event1data))
		require.NoError(t, ps.Publish(ctx, channel2name, event2data))

		var (
			event1, isSub1open = <-sub1
			event2, isSub2open = <-sub2
		)

		require.Equal(t, event1data, event1)
		require.True(t, isSub1open)

		require.Equal(t, event2data, event2)
		require.True(t, isSub2open)
	})

	require.NoError(t, ps.Publish(ctx, channel1name, event1data)) // will not be delivered
	require.NoError(t, ps.Publish(ctx, channel2name, event2data)) // will not be delivered

	close1()
	close2()

	require.NoError(t, ps.Publish(ctx, channel1name, event1data)) // will not be delivered
	require.NoError(t, ps.Publish(ctx, channel2name, event2data)) // will not be delivered

	t.Run("read from closed", func(t *testing.T) {
		var (
			event1, isSub1open = <-sub1
			event2, isSub2open = <-sub2
		)

		require.Empty(t, event1)
		require.False(t, isSub1open)

		require.Empty(t, event2)
		require.False(t, isSub2open)
	})

	t.Run("publish into non-existing channel", func(t *testing.T) {
		require.NoError(t, ps.Publish(ctx, "baz", "event3"))
	})
}

func testRaceProvocation(t *testing.T, new func() pubSub[any]) {
	t.Helper()

	var (
		ps  = new()
		ctx = context.Background()
		wg  sync.WaitGroup
	)

	const channelName, eventData = "foo", "event"

	for range 1_000 {
		sub, unsubscribe, err := ps.Subscribe(ctx, channelName) // subscribe
		require.NoError(t, err)

		wg.Add(1)

		go func() {
			defer wg.Done()

			require.Equal(t, <-sub, eventData) // receive (block until event is received)

			unsubscribe() // unsubscribe
		}()

		wg.Add(1)

		go func() {
			defer wg.Done()

			require.NoError(t, ps.Publish(ctx, channelName, eventData)) // publish
		}()
	}

	wg.Wait()
}

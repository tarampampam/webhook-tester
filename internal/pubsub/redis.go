package pubsub

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v2/internal/encoding"
)

type redisClient interface {
	redis.Cmdable
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

type Redis[T any] struct {
	client redisClient
	encDec encoding.EncoderDecoder
}

var ( // ensure interface implementation
	_ Publisher[any]  = (*Redis[any])(nil)
	_ Subscriber[any] = (*Redis[any])(nil)
)

func NewRedis[T any](c redisClient, encDec encoding.EncoderDecoder) *Redis[T] {
	return &Redis[T]{client: c, encDec: encDec}
}

func (ps *Redis[T]) Subscribe(ctx context.Context, topic string) (_ <-chan T, unsubscribe func(), _ error) {
	var (
		pubSub        = ps.client.Subscribe(ctx, topic)
		sub           = make(chan T)
		stop, stopped = make(chan struct{}), make(chan struct{})
	)

	go func() {
		defer close(stopped) // notify unsubscribe that the goroutine is stopped

		var channel = pubSub.Channel() // get the channel for the topic

		defer func() { _ = pubSub.Close() }() // guaranty that pubSub will be closed

		for {
			select {
			case <-ctx.Done():
				return // check the context
			case <-stop:
				return // check the stopping notification
			case msg := <-channel: // wait for the message
				if msg == nil {
					continue
				}

				var event T

				if err := ps.encDec.Decode([]byte(msg.Payload), &event); err != nil {
					continue
				}

				select { // send the event to the subscriber
				case <-ctx.Done():
					return
				case <-stop:
					return
				case sub <- event:
				}
			}
		}
	}()

	return sub, sync.OnceFunc(func() {
		_ = pubSub.Close() // close the subscription

		close(stop) // notify the goroutine to stop

		<-stopped // wait for the goroutine to stop

		close(sub) // close the subscription channel
	}), nil
}

func (ps *Redis[T]) Publish(ctx context.Context, topic string, event T) error {
	data, mErr := ps.encDec.Encode(event)
	if mErr != nil {
		return mErr
	}

	return ps.client.Publish(ctx, topic, data).Err()
}

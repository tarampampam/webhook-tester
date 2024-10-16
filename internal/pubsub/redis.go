package pubsub

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/redis/go-redis/v9"
)

type redisClient interface {
	redis.Cmdable
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

type Redis[T any] struct {
	client redisClient
}

var ( // ensure interface implementation
	_ Publisher[any]  = (*Redis[any])(nil)
	_ Subscriber[any] = (*Redis[any])(nil)
)

func NewRedis[T any](client redisClient) *Redis[T] { return &Redis[T]{client: client} }

func (*Redis[T]) unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (*Redis[T]) marshal(v any) ([]byte, error)      { return json.Marshal(v) }

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

				if err := ps.unmarshal([]byte(msg.Payload), &event); err != nil {
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
	data, mErr := ps.marshal(event)
	if mErr != nil {
		return mErr
	}

	return ps.client.Publish(ctx, topic, data).Err()
}

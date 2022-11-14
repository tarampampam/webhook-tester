package pubsub

import (
	"context"
	"errors"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

type (
	// Redis publisher/subscriber uses redis server for events publishing and delivering to the subscribers. Useful for
	// application "distributed" mode running.
	//
	// Publishing/subscribing events order and delivering (in cases then there is no one active subscriber for the
	// channel) is NOT guaranteed.
	//
	// Node: Do not forget to Close it after all. Closed publisher/subscriber cannot be opened back.
	Redis struct {
		ctx context.Context
		rdb *redis.Client

		subsMu sync.Mutex
		subs   map[string]*redisSubscription

		closedMu sync.Mutex
		closed   bool
	}

	redisSubscription struct {
		start sync.Once
		stop  chan struct{}

		subscribersMu sync.Mutex
		subscribers   map[chan<- Event]struct{}
	}
)

// NewRedis creates new redis publisher/subscriber.
func NewRedis(ctx context.Context, rdb *redis.Client) *Redis {
	return &Redis{
		ctx:  ctx,
		rdb:  rdb,
		subs: make(map[string]*redisSubscription),
	}
}

// redisEvent is an internal structure for events serialization.
type redisEvent struct {
	Name string `msgpack:"n"`
	Data []byte `msgpack:"d"`
}

// Publish an event into passed channel.
func (ps *Redis) Publish(channelName string, event Event) error {
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	b, err := msgpack.Marshal(redisEvent{Name: event.Name(), Data: event.Data()})
	if err != nil {
		return err
	}

	return ps.rdb.Publish(ps.ctx, channelName, string(b)).Err()
}

// Subscribe to the named channel and receive Event's into the passed channel.
//
// Note: that this function does not wait on a response from redis server, so the subscription may not be active
// immediately.
func (ps *Redis) Subscribe(channelName string, channel chan<- Event) error { //nolint:funlen
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	// create subscription if needed
	ps.subsMu.Lock()
	if _, exists := ps.subs[channelName]; !exists {
		ps.subs[channelName] = &redisSubscription{
			stop:        make(chan struct{}, 1),
			subscribers: make(map[chan<- Event]struct{}),
		}
	}
	ps.subsMu.Unlock()

	ps.subs[channelName].subscribersMu.Lock()
	defer ps.subs[channelName].subscribersMu.Unlock()

	// append passed channel into subscribers map
	if _, exists := ps.subs[channelName].subscribers[channel]; exists {
		return errors.New("already subscribed")
	}

	ps.subs[channelName].subscribers[channel] = struct{}{}

	ps.subs[channelName].start.Do(func() {
		started := make(chan struct{}, 1)

		go func(sub *redisSubscription) {
			var (
				pubSub = ps.rdb.Subscribe(ps.ctx, channelName)
				ch     = pubSub.Channel()
			)

			defer func() {
				_ = pubSub.Close()
				_ = pubSub.Unsubscribe(ps.ctx, channelName)
			}()

			started <- struct{}{}
			close(started)

			for {
				select {
				case <-sub.stop:
					return

				case msg, opened := <-ch:
					if !opened {
						return
					}

					var rawEvent redisEvent
					if err := msgpack.Unmarshal([]byte(msg.Payload), &rawEvent); err != nil {
						continue
					}

					e := event{name: rawEvent.Name, data: rawEvent.Data}

					sub.subscribersMu.Lock()

					for receiver := range sub.subscribers { // iterate over all subscribed channels
						go func(target chan<- Event) {
							target <- &e // <- panic can be occurred here (if channel was closed too early from outside)
						}(receiver)
					}

					sub.subscribersMu.Unlock()
				}
			}
		}(ps.subs[channelName])

		<-started // make sure that subscription was started
	})

	return nil
}

// Unsubscribe the subscription to the named channel for the passed events channel. Be careful with channel closing,
// this can call the panics if some Event's scheduled for publishing.
func (ps *Redis) Unsubscribe(channelName string, channel chan Event) error {
	if channelName == "" {
		return errors.New("empty channel name is not allowed")
	}

	if ps.isClosed() {
		return errors.New("closed")
	}

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if _, exists := ps.subs[channelName]; !exists {
		return errors.New("subscription does not exists")
	}

	if _, exists := ps.subs[channelName].subscribers[channel]; !exists {
		return errors.New("channel was not subscribed")
	}

	// cancel subscription
	ps.subs[channelName].subscribersMu.Lock()
	delete(ps.subs[channelName].subscribers, channel)
	subscribersCount := len(ps.subs[channelName].subscribers)
	ps.subs[channelName].subscribersMu.Unlock()

	// in case when there is no one active subscriber for the channel - we should to notify redis subscriber about
	// stopping and clean up
	if subscribersCount == 0 {
		ps.subs[channelName].stop <- struct{}{}
		close(ps.subs[channelName].stop)
		delete(ps.subs, channelName)
	}

	return nil
}

func (ps *Redis) isClosed() (isClosed bool) {
	ps.closedMu.Lock()
	isClosed = ps.closed
	ps.closedMu.Unlock()

	return
}

// Close this publisher/subscriber. This function can be called only once.
func (ps *Redis) Close() error {
	if ps.isClosed() {
		return errors.New("already closed")
	}

	ps.closedMu.Lock()
	ps.closed = true
	ps.closedMu.Unlock()

	ps.subsMu.Lock()
	for channelName, sub := range ps.subs {
		sub.stop <- struct{}{}
		close(sub.stop)
		delete(ps.subs, channelName)
	}
	ps.subsMu.Unlock()

	return nil
}

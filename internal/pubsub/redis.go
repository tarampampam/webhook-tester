package pubsub

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 is used for non-cryptographic hashing of topic names to Redis channel keys
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisPubSubPrefix = "wht:v3:pubsub:"

// Redis is a Redis-backed pub/sub using the Redis Pub/Sub command set.
type Redis struct {
	client redis.UniversalClient

	subsMu sync.Mutex // guards subs
	subs   map[chan RequestEvent]*redisSubState

	closed atomic.Bool
}

var ( // compile-time interface assertions
	_ PubSub    = (*Redis)(nil)
	_ io.Closer = (*Redis)(nil)
)

// NewRedis creates a new Redis-backed pub/sub.
func NewRedis(client redis.UniversalClient) *Redis {
	return &Redis{client: client, subs: make(map[chan RequestEvent]*redisSubState)}
}

// Publish implements [Publisher].
func (ps *Redis) Publish(ctx context.Context, topic string, event RequestEvent) error {
	if err, cl := ctx.Err(), ps.closed.Load(); err != nil || cl {
		if cl {
			err = ErrClosed
		}

		return err
	}

	data, err := json.Marshal(requestEventToRedisEvent(event))
	if err != nil {
		return err
	}

	return ps.client.Publish(ctx, redisPubSubPrefix+ps.hash(topic), data).Err()
}

// Subscribe implements [Subscriber].
func (ps *Redis) Subscribe(ctx context.Context, topic string) (<-chan RequestEvent, func(), error) {
	// check for the context cancellation and closed state before acquiring the lock
	if err, cl := ctx.Err(), ps.closed.Load(); err != nil || cl {
		if cl {
			err = ErrClosed
		}

		return nil, func() {}, err
	}

	// acquire the lock to create the subscription
	ps.subsMu.Lock()

	// check for closed state again after acquiring the lock to prevent a race with Close
	if ps.closed.Load() {
		ps.subsMu.Unlock()

		return nil, func() {}, ErrClosed
	}

	var (
		subCh   = make(chan RequestEvent)
		stop    = make(chan struct{})
		stopped = make(chan struct{})
	)

	redisSub := ps.client.Subscribe(ctx, redisPubSubPrefix+ps.hash(topic))

	unsub := sync.OnceFunc(func() {
		ps.subsMu.Lock()
		delete(ps.subs, subCh)
		ps.subsMu.Unlock()

		_ = redisSub.Close() // unsubscribe from Redis; causes redisCh to close

		close(stop)  // unblock the goroutine if it is stuck in the inner send select
		<-stopped    // wait for the goroutine to exit
		close(subCh) // safe: no more senders
	})

	ps.subs[subCh] = &redisSubState{unsub}
	ps.subsMu.Unlock()

	go func() {
		defer close(stopped)

		redisCh := redisSub.Channel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case msg, ok := <-redisCh:
				if !ok || msg == nil {
					return
				}

				var re redisEvent
				if err := json.Unmarshal([]byte(msg.Payload), &re); err != nil {
					continue // silently ignore malformed messages
				}

				event := re.toRequestEvent()

				select {
				case <-ctx.Done():
					return
				case <-stop:
					return
				case subCh <- event:
				}
			}
		}
	}()

	// auto-unsubscribe when the caller's context is canceled or the goroutine exits unexpectedly
	go func() {
		select {
		case <-ctx.Done():
			unsub()
		case <-stopped: // redisCh closed unexpectedly; ensures ch is always closed
			unsub()
		case <-stop: // already unsubscribed manually; exit to avoid goroutine leak
		}
	}()

	return subCh, unsub, nil
}

// Close implements [io.Closer]. It unsubscribes all active subscribers, releases their channels and goroutines,
// and marks the instance as closed. Any subsequent call to Publish or Subscribe returns [ErrClosed].
// Close itself is idempotent - a second call returns [ErrClosed] without blocking.
func (ps *Redis) Close() error {
	if !ps.closed.CompareAndSwap(false, true) {
		return ErrClosed
	}

	ps.subsMu.Lock()

	unsubs := make([]func(), 0, len(ps.subs))
	for _, state := range ps.subs {
		unsubs = append(unsubs, state.unsub)
	}

	clear(ps.subs)
	ps.subs = make(map[chan RequestEvent]*redisSubState)
	ps.subsMu.Unlock()

	for _, unsub := range unsubs {
		unsub()
	}

	return nil
}

// hash returns the lowercase hex-encoded MD5 of the input string.
// Used for generating Redis channel names from topic strings.
func (*Redis) hash(v string) string { h := md5.Sum([]byte(v)); return hex.EncodeToString(h[:]) } //nolint:nlreturn,gosec

// --------------------------------------------------------------------------------------------------------------------

type redisSubState struct {
	unsub func()
}

type redisEvent struct {
	Action  RequestAction      `json:"action"`
	Request *redisEventRequest `json:"request,omitempty"`
}

type redisEventRequest struct {
	ID          string             `json:"id"`
	ClientAddr  string             `json:"client_addr,omitempty"`
	Method      string             `json:"method,omitempty"`
	Headers     []redisEventHeader `json:"headers,omitempty"`
	URL         string             `json:"url,omitempty"`
	CreatedAtUs int64              `json:"created_at_us"`
}

type redisEventHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// toRequestEvent converts a redisEvent to a [RequestEvent].
func (e redisEvent) toRequestEvent() RequestEvent {
	out := RequestEvent{Action: e.Action}
	if e.Request != nil {
		rd := &RequestData{
			ID:         e.Request.ID,
			ClientAddr: e.Request.ClientAddr,
			Method:     e.Request.Method,
			URL:        e.Request.URL,
			CreatedAt:  time.UnixMicro(e.Request.CreatedAtUs).UTC(),
		}

		if len(e.Request.Headers) > 0 {
			rd.Headers = make([]HttpHeader, len(e.Request.Headers))
			for i, h := range e.Request.Headers {
				rd.Headers[i] = HttpHeader(h)
			}
		}

		out.Request = rd
	}

	return out
}

// requestEventToRedisEvent converts a [RequestEvent] to a redisEvent.
func requestEventToRedisEvent(in RequestEvent) redisEvent {
	out := redisEvent{Action: in.Action}
	if r := in.Request; r != nil {
		rd := &redisEventRequest{
			ID:          r.ID,
			ClientAddr:  r.ClientAddr,
			Method:      r.Method,
			URL:         r.URL,
			CreatedAtUs: r.CreatedAt.UnixMicro(),
		}

		if len(r.Headers) > 0 {
			rd.Headers = make([]redisEventHeader, len(r.Headers))
			for i, h := range r.Headers {
				rd.Headers[i] = redisEventHeader(h)
			}
		}

		out.Request = rd
	}

	return out
}

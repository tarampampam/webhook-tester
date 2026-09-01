package pubsub_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
)

func redisFactory(tb testing.TB) pubsub.PubSub {
	tb.Helper()

	srv := miniredis.RunT(tb)

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})

	tb.Cleanup(func() { _ = client.Close() })

	ps := pubsub.NewRedis(client)

	tb.Cleanup(func() { _ = ps.Close() })

	return ps
}

func TestRedis(t *testing.T) { t.Parallel(); RunSuite(t, redisFactory) }

func BenchmarkRedis(b *testing.B) { RunBenchmarks(b, redisFactory) }

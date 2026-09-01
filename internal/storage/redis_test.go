package storage_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

func redisFactory(tb testing.TB, limit uint, timeNow func() time.Time) storage.Storage {
	tb.Helper()

	srv := miniredis.RunT(tb)

	initial := timeNow()
	srv.SetTime(initial)

	syncer := &miniredisTimeSyncer{fakeNow: timeNow, srv: srv, lastSeen: initial}

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	client.AddHook(syncer)

	r := storage.NewRedis(client, limit, storage.WithRedisTimeNow(syncer.Now))

	tb.Cleanup(func() { _ = client.Close() })

	return r
}

func TestRedis(t *testing.T) {
	t.Parallel()

	RunSuite(t, redisFactory)
}

func BenchmarkRedis(b *testing.B) { RunBenchmarks(b, redisFactory) }

// --------------------------------------------------------------------------------------------------------------------

// miniredisTimeSyncer bridges the shared test suite's fake clock with miniredis.
//
// It implements redis.Hook so that miniredis advances before every Redis command, making expiry tests work without
// any time.Sleep or real TTL waits.
//
// It also implements the timeNow signature so that display timestamps (CreatedAt, ExpiresAt) in storage metadata
// reflect the fake clock rather than wall-clock time.
type miniredisTimeSyncer struct {
	fakeNow  func() time.Time
	srv      *miniredis.Miniredis
	mu       sync.Mutex
	lastSeen time.Time
}

func (m *miniredisTimeSyncer) sync() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.fakeNow()
	if now.After(m.lastSeen) {
		m.srv.SetTime(now)
		m.srv.FastForward(now.Sub(m.lastSeen))
		m.lastSeen = now
	}
}

// Now is passed to storage.WithRedisTimeNow so that display timestamps use fake time.
func (m *miniredisTimeSyncer) Now(_ context.Context) (time.Time, error) {
	m.sync()

	return m.fakeNow(), nil
}

// DialHook implements [redis.Hook].
func (m *miniredisTimeSyncer) DialHook(next redis.DialHook) redis.DialHook { return next }

// ProcessHook implements [redis.Hook] - syncs miniredis before every single command.
func (m *miniredisTimeSyncer) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		m.sync()

		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements [redis.Hook] - syncs miniredis before every pipeline.
func (m *miniredisTimeSyncer) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		m.sync()

		return next(ctx, cmds)
	}
}

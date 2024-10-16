package storage_test

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func TestRedis_Session_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var mini = miniredis.RunT(t)

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewRedis(
				redis.NewClient(&redis.Options{Addr: mini.Addr()}),
				sTTL,
				maxReq,
			)
		},
		func(t time.Duration) { mini.FastForward(t); <-time.After(t) },
	)
}

func TestRedis_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var mini = miniredis.RunT(t)

	testRequestCreateReadDelete(t, func(sTTL time.Duration, maxReq uint32) storage.Storage {
		return storage.NewRedis(
			redis.NewClient(&redis.Options{Addr: mini.Addr()}),
			sTTL,
			maxReq,
		)
	})
}

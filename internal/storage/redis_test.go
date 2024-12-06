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

	var (
		mini = miniredis.RunT(t)
		ft   = newFakeTime(t)
	)

	testSessionCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewRedis(
				redis.NewClient(&redis.Options{Addr: mini.Addr()}),
				sTTL,
				maxReq,
				storage.WithRedisTimeNow(ft.Get),
			)
		},
		func(t time.Duration) { mini.FastForward(t); ft.Add(t) },
	)
}

func TestRedis_Request_CreateReadDelete(t *testing.T) {
	t.Parallel()

	var (
		mini = miniredis.RunT(t)
		ft   = newFakeTime(t)
	)

	testRequestCreateReadDelete(t,
		func(sTTL time.Duration, maxReq uint32) storage.Storage {
			return storage.NewRedis(
				redis.NewClient(&redis.Options{Addr: mini.Addr()}),
				sTTL,
				maxReq,
				storage.WithRedisTimeNow(ft.Get),
			)
		},
		func(t time.Duration) { mini.FastForward(t); ft.Add(t) },
	)
}

//	func TestRedis_RaceProvocation(t *testing.T) {
//		t.Parallel()
//
//		var mini = miniredis.RunT(t)
//
//		testRaceProvocation(t, func(sTTL time.Duration, maxReq uint32) storage.Storage {
//			return storage.NewRedis(
//				redis.NewClient(&redis.Options{Addr: mini.Addr()}),
//				encDec,
//				sTTL,
//				maxReq,
//			)
//		})
//	}

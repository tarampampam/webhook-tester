package pubsub_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
)

func TestRedis_Publish_and_Receive(t *testing.T) {
	t.Parallel()

	var mini = miniredis.RunT(t)

	testPublishAndReceive(t, func() pubSub[any] {
		return pubsub.NewRedis[any](
			redis.NewClient(&redis.Options{Addr: mini.Addr()}),
			encDec,
		)
	})
}

//	func TestRedis_RaceProvocation(t *testing.T) {
//		t.Parallel()
//
//		var mini = miniredis.RunT(t)
//
//		testRaceProvocation(t, func() pubSub[any] {
//			return pubsub.NewRedis[any](
//				redis.NewClient(&redis.Options{Addr: mini.Addr()}),
//				encDec,
//			)
//		})
//	}

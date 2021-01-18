package checkers

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestReadyChecker_CheckWithoutRedisClient(t *testing.T) {
	assert.NoError(t, NewReadyChecker(context.Background(), nil).Check())
}

func TestReadyChecker_CheckSuccessWithRedisClient(t *testing.T) {
	// start mini-redis
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer rdb.Close()

	assert.NoError(t, NewReadyChecker(context.Background(), rdb).Check())
}

func TestReadyChecker_CheckFailedWithRedisClient(t *testing.T) {
	// start mini-redis
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer rdb.Close()

	mini.SetError("foo err")
	assert.Error(t, NewReadyChecker(context.Background(), rdb).Check())
	mini.SetError("")
}

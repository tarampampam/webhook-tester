package broadcast_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
)

func TestNone_Publish(t *testing.T) {
	var none broadcast.None

	var (
		counter1, counter2 int
		ourEvent           = broadcast.NewRequestRegisteredEvent("bar")
	)

	none.OnPublish(func(ch string, e broadcast.Event) {
		counter1++

		assert.Equal(t, "foo", ch)
		assert.Same(t, e, ourEvent)
	})

	none.OnPublish(func(ch string, e broadcast.Event) {
		counter2++

		assert.Equal(t, "foo", ch)
		assert.Same(t, e, ourEvent)
	})

	assert.NoError(t, none.Publish("foo", ourEvent))
	assert.Equal(t, 1, counter1)
	assert.Equal(t, 1, counter2)
	assert.NoError(t, none.Publish("foo", ourEvent))
	assert.Equal(t, 2, counter1)
	assert.Equal(t, 2, counter2)
}

func TestNone_Concurrent(t *testing.T) {
	var none broadcast.None

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			none.OnPublish(func(ch string, e broadcast.Event) {
				timer := time.NewTimer(time.Microsecond)
				<-timer.C
				timer.Stop()
			})

			wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			assert.NoError(t, none.Publish("foo", broadcast.NewRequestRegisteredEvent("bar")))

			wg.Done()
		}()
	}

	wg.Wait()
}

package broadcast

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNone_Publish(t *testing.T) {
	var none None

	var (
		counter1, counter2 int
		ourEvent           = NewRequestRegisteredEvent("bar")
	)

	none.OnPublish(func(ch string, e Event) {
		counter1++

		assert.Equal(t, "foo", ch)
		assert.Same(t, e, ourEvent)
	})

	none.OnPublish(func(ch string, e Event) {
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
	var none None

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			none.OnPublish(func(ch string, e Event) {
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
			assert.NoError(t, none.Publish("foo", NewRequestRegisteredEvent("bar")))

			wg.Done()
		}()
	}

	wg.Wait()
}

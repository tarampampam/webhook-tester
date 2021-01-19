package broadcast

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNone_Publish(t *testing.T) {
	none := None{}

	// basic event publishing
	ch, last := none.LastPublishedEvent()
	assert.Nil(t, last)
	assert.Empty(t, ch)

	e := NewRequestRegisteredEvent("bar")
	assert.NoError(t, none.Publish("foo", e))

	// get last recorded event
	ch, last = none.LastPublishedEvent()
	assert.Equal(t, "foo", ch)
	assert.Same(t, e, last)

	// set some error
	none.SetError(errors.New("foo error"))
	assert.EqualError(t, none.Publish("aaa", e), "foo error")

	// event must be recorded anyway
	ch, last = none.LastPublishedEvent()
	assert.Equal(t, "aaa", ch)
	assert.Same(t, e, last)

	// unset the error
	none.SetError(nil)

	// and then all works fine again
	assert.NoError(t, none.Publish("foo", e))

	// and event recorded successful
	ch, last = none.LastPublishedEvent()
	assert.Equal(t, "foo", ch)
	assert.Same(t, e, last)
}

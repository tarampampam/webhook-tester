package null

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage_All(t *testing.T) {
	b := Broadcaster{
		Error: errors.New(""),
	}

	assert.Same(t, b.Error, b.Publish("foo", "bar", 123))
	assert.Equal(t, "foo", b.GetLastPublishedChannel())
	assert.Equal(t, "bar", b.GetLastPublishedEventName())
	assert.Equal(t, 123, b.GetLastPublishedData())
}

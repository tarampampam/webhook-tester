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

	assert.Same(t, b.Error, b.Publish("", "", nil))
}

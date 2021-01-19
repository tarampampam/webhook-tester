package checkers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveChecker_Check(t *testing.T) {
	assert.NoError(t, NewLiveChecker().Check())
}

package checkers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/checkers"
)

func TestLiveChecker_Check(t *testing.T) {
	assert.NoError(t, checkers.NewLiveChecker().Check())
}

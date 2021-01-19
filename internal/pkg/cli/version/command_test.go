package version

import (
	"runtime"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func TestProperties(t *testing.T) {
	cmd := NewCommand("")

	assert.Equal(t, "version", cmd.Use)
	assert.ElementsMatch(t, []string{"v", "ver"}, cmd.Aliases)
	assert.NotNil(t, cmd.RunE)
}

func TestCommandRun(t *testing.T) {
	cmd := NewCommand("1.2.3@foobar")
	cmd.SetArgs([]string{})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Contains(t, output, "1.2.3@foobar")
	assert.Contains(t, output, runtime.Version())
}

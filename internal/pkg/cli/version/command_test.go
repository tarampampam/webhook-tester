package version

import (
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func TestCommand_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		giveArgs         []string
		wantOutput       []string
		wantErr          bool
		wantErrorMessage string
	}{
		{
			name:             "By default",
			giveArgs:         []string{},
			wantOutput:       []string{"Version:", "undefined@undefined", "\n"},
			wantErr:          false,
			wantErrorMessage: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var cmd = Command{}

			assert.NoError(t, cmd.Execute(tt.giveArgs))

			output := capturer.CaptureStdout(func() {
				err = cmd.Execute(tt.giveArgs)
			})

			if tt.wantOutput != nil {
				for _, line := range tt.wantOutput {
					assert.Contains(t, output, line)
				}
			}

			if tt.wantErr && err.Error() != tt.wantErrorMessage {
				t.Errorf("Expected error message [%s] was not found in %v", tt.wantErrorMessage, err)
			}
		})
	}
}

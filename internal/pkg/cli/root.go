package cli

import (
	"github.com/tarampampam/webhook-tester/internal/pkg/cli/serve"
	"github.com/tarampampam/webhook-tester/internal/pkg/cli/version"
)

// Root is a basic commands struct.
type Root struct {
	Version version.Command `command:"version" alias:"v" description:"Display application version"`
	Serve   serve.Command   `command:"serve" alias:"s" description:"Start application web server"`
}

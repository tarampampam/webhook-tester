package cmd

import (
	"webhook-tester/cmd/serve"
	"webhook-tester/cmd/version"
)

// Root is a basic commands struct.
type Root struct {
	Version version.Command `command:"version" alias:"v" description:"Display application version"`
	Serve   serve.Command   `command:"serve" alias:"s" description:"Start application web server"`
}

package shared

import (
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/webhook-tester/internal/env"
)

var PortNumberFlag = &cli.UintFlag{ //nolint:gochecknoglobals
	Name:    "port",
	Aliases: []string{"p"},
	Usage:   "Server TCP port number",
	Value:   8080, //nolint:gomnd
	EnvVars: []string{env.ListenPort.String()},
}

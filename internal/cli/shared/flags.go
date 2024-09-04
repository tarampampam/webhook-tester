package shared

import (
	"github.com/urfave/cli/v2"

	"gh.tarampamp.am/webhook-tester/internal/env"
)

var PortNumberFlag = &cli.UintFlag{
	Name:    "port",
	Aliases: []string{"p"},
	Usage:   "Server TCP port number",
	Value:   8080,
	EnvVars: []string{env.ListenPort.String(), env.Port.String()},
}

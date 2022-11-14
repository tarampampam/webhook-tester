// Package healthcheck contains CLI `healthcheck` command implementation.
package healthcheck

import (
	"errors"
	"math"

	"github.com/urfave/cli/v2"

	"github.com/tarampampam/webhook-tester/internal/cli/shared"
)

type checker interface {
	Check(port uint16) error
}

// NewCommand creates `healthcheck` command.
func NewCommand(checker checker) *cli.Command {
	return &cli.Command{
		Name:    "healthcheck",
		Aliases: []string{"chk", "health", "check"},
		Usage:   "Health checker for the HTTP server. Use case - docker healthcheck",
		Action: func(c *cli.Context) error {
			var port = c.Uint(shared.PortNumberFlag.Name)

			if port > math.MaxUint16 {
				return errors.New("wrong TCP port number")
			}

			return checker.Check(uint16(port))
		},
		Flags: []cli.Flag{
			shared.PortNumberFlag,
		},
	}
}

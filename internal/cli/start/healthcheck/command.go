package healthcheck

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// NewCommand creates `healthcheck` command.
func NewCommand(defaultHttpPort uint16) *cli.Command {
	var (
		httpPortFlag = cli.UintFlag{
			Name:     "http-port",
			Usage:    "HTTP server port",
			Value:    uint64(defaultHttpPort),
			Sources:  cli.EnvVars("HTTP_PORT"),
			OnlyOnce: true,
			Validator: func(port uint64) error {
				if port == 0 || port > 65535 {
					return fmt.Errorf("wrong TCP port number [%d]", port)
				}

				return nil
			},
		}
	)

	return &cli.Command{
		Name:    "healthcheck",
		Aliases: []string{"hc", "health", "check"},
		Usage:   "Health checker for the HTTP(S) servers. Use case - docker healthcheck",
		Action: func(ctx context.Context, c *cli.Command) error {
			return NewHealthChecker().Check(ctx, uint(c.Uint(httpPortFlag.Name)))
		},
		Flags: []cli.Flag{
			&httpPortFlag,
		},
	}
}

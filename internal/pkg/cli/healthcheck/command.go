// Package healthcheck contains CLI `healthcheck` command implementation.
package healthcheck

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tarampampam/webhook-tester/internal/pkg/env"
)

type checker interface {
	Check(port uint16) error
}

// NewCommand creates `healthcheck` command.
func NewCommand(checker checker) *cobra.Command {
	var port uint16

	cmd := &cobra.Command{
		Use:     "healthcheck",
		Aliases: []string{"chk", "health", "check"},
		Short:   "Health checker for the HTTP server. Use case - docker healthcheck.",
		Hidden:  true,
		PreRunE: func(*cobra.Command, []string) error {
			if envPort, exists := env.ListenPort.Lookup(); exists && envPort != "" {
				if p, err := strconv.ParseUint(envPort, 10, 16); err == nil {
					port = uint16(p)
				} else {
					return fmt.Errorf("wrong TCP port environment variable [%s] value", envPort)
				}
			}

			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			return checker.Check(port)
		},
	}

	cmd.Flags().Uint16VarP(
		&port,
		"port",
		"p",
		8080,
		fmt.Sprintf("TCP port number [$%s]", env.ListenPort),
	)

	return cmd
}

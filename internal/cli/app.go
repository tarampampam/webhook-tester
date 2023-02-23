package cli

import (
	"context"
	"fmt"
	"runtime"

	"github.com/urfave/cli/v2"

	"gh.tarampamp.am/webhook-tester/internal/checkers"
	"gh.tarampamp.am/webhook-tester/internal/cli/healthcheck"
	"gh.tarampamp.am/webhook-tester/internal/cli/serve"
	"gh.tarampamp.am/webhook-tester/internal/logger"
	"gh.tarampamp.am/webhook-tester/internal/version"
)

// NewApp creates new console application.
func NewApp() *cli.App {
	const (
		verboseFlagName = "verbose"
		debugFlagName   = "debug"
		logJSONFlagName = "log-json"
	)

	const loggingCategoryName = "Logging"

	// create "default" logger (will be overwritten later with customized)
	log, err := logger.New(false, false, false)
	if err != nil {
		panic(err)
	}

	return &cli.App{
		Usage: "CLI client for images compressing using tinypng.com API",
		Before: func(c *cli.Context) error {
			_ = log.Sync() // sync previous logger instance

			customizedLog, e := logger.New(c.Bool(verboseFlagName), c.Bool(debugFlagName), c.Bool(logJSONFlagName))
			if e != nil {
				return e
			}

			*log = *customizedLog // override "default" logger with customized

			return nil
		},
		Commands: []*cli.Command{
			healthcheck.NewCommand(checkers.NewHealthChecker(context.Background())),
			serve.NewCommand(log),
		},
		Version: fmt.Sprintf("%s (%s)", version.Version(), runtime.Version()),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     verboseFlagName,
				Category: loggingCategoryName,
				Usage:    "verbose output",
			},
			&cli.BoolFlag{
				Name:     debugFlagName,
				Category: loggingCategoryName,
				Usage:    "debug output",
			},
			&cli.BoolFlag{
				Name:     logJSONFlagName,
				Category: loggingCategoryName,
				Usage:    "logs in JSON format",
			},
		},
	}
}

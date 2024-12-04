package cli

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"

	"gh.tarampamp.am/webhook-tester/v2/internal/cli/start"
	"gh.tarampamp.am/webhook-tester/v2/internal/logger"
	"gh.tarampamp.am/webhook-tester/v2/internal/version"
)

//go:generate go run app_readme.go

// NewApp creates new console application.
func NewApp() *cli.Command { //nolint:funlen
	var (
		logLevelFlag = cli.StringFlag{
			Name:     "log-level",
			Value:    logger.InfoLevel.String(),
			Usage:    "Logging level (" + strings.Join(logger.LevelStrings(), "/") + ")",
			Sources:  cli.EnvVars("LOG_LEVEL"),
			OnlyOnce: true,
			Config:   cli.StringConfig{TrimSpace: true},
			Validator: func(s string) error {
				if _, err := logger.ParseLevel(s); err != nil {
					return err
				}

				return nil
			},
		}

		logFormatFlag = cli.StringFlag{
			Name:     "log-format",
			Value:    logger.ConsoleFormat.String(),
			Usage:    "Logging format (" + strings.Join(logger.FormatStrings(), "/") + ")",
			Sources:  cli.EnvVars("LOG_FORMAT"),
			OnlyOnce: true,
			Config:   cli.StringConfig{TrimSpace: true},
			Validator: func(s string) error {
				if _, err := logger.ParseFormat(s); err != nil {
					return err
				}

				return nil
			},
		}
	)

	// create "default" logger (will be overwritten later with customized)
	var log, _ = logger.New(logger.InfoLevel, logger.ConsoleFormat) // error will never occur

	const defaultHttpPort uint16 = 8080

	return &cli.Command{
		Usage: "webhook tester",
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			_ = log.Sync() // sync previous logger instance

			var (
				logLevel, _  = logger.ParseLevel(c.String(logLevelFlag.Name))   // error ignored because the flag validates itself
				logFormat, _ = logger.ParseFormat(c.String(logFormatFlag.Name)) // --//--
			)

			configured, err := logger.New(logLevel, logFormat) // create new logger instance
			if err != nil {
				return ctx, err
			}

			*log = *configured // replace "default" logger with customized

			return ctx, nil
		},
		Commands: []*cli.Command{
			start.NewCommand(log, defaultHttpPort),
		},
		Version: fmt.Sprintf("%s (%s)", version.Version(), runtime.Version()),
		Flags: []cli.Flag{ // global flags
			&logLevelFlag,
			&logFormatFlag,
		},
	}
}

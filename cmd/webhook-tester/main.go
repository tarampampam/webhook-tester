package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/tarampampam/webhook-tester/internal/pkg/cli"
)

func main() {
	// parse the arguments
	if _, err := flags.NewParser(&cli.Root{}, flags.Default).Parse(); err != nil {
		// make error type checking
		if e, ok := err.(*flags.Error); (ok && e.Type != flags.ErrHelp) || !ok {
			os.Exit(1)
		}
	}
}

package main

import (
	"os"
	"webhook-tester/cmd"

	"github.com/jessevdk/go-flags"
)

func main() {
	// parse the arguments
	if _, err := flags.NewParser(&cmd.Root{}, flags.Default).Parse(); err != nil {
		// make error type checking
		if e, ok := err.(*flags.Error); (ok && e.Type != flags.ErrHelp) || !ok {
			os.Exit(1)
		}
	}
}

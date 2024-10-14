package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"gh.tarampamp.am/webhook-tester/internal/cli"
)

// main CLI application entrypoint.
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())

		os.Exit(1)
	}
}

// run is the entry point of the program. The code is in separate function to allow executing deferred functions
// before exiting (os.Exit does not execute deferred functions).
func run() error {
	defer runtime.Gosched() // increase the chance of running deferred functions before exiting

	var ctx, cancel = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return cli.NewApp().Run(ctx, os.Args)
}

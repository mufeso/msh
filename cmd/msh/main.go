package main

import (
	"context"
	"os"

	"example.com/msh/internal/app"
	"example.com/msh/internal/signals"
)

func main() {
	os.Exit(run())
}

// run is separate from main so that deferred functions execute before
// os.Exit (os.Exit does not run defers).
func run() int {
	ctx, stop := signals.Context(context.Background(), os.Stderr)
	defer stop()
	return app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}

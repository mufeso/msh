// Command msh is "My Parallel SSH", a command-line SSH fleet-management
// tool for Linux servers.
//
// This package is intentionally tiny: it connects the real process (its
// arguments, stdout, stderr and exit code) to the testable app package.
package main

import (
	"os"

	"example.com/msh/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}

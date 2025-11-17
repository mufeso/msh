// Command msh is "My Parallel SSH", a command-line SSH fleet-management
// tool for Linux servers.
package main

import (
	"fmt"
	"io"
	"os"
)

// Application identity.
const (
	appName     = "msh"
	appLongName = "My Parallel SSH"
)

// version is a variable rather than a constant because a later step sets it
// at build time with: go build -ldflags "-X ...=1.2.3".
var version = "0.1.0-dev"

// Exit codes.
const (
	exitOK    = 0
	exitUsage = 2
)

// banner returns the application's display name.
func banner() string {
	return appName + " — " + appLongName
}

// versionLine returns the text printed by --version.
func versionLine() string {
	return fmt.Sprintf("%s version %s", appName, version)
}

// run is the testable entry point. It returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, banner())
		return exitOK
	}

	switch args[0] {
	case "--version":
		fmt.Fprintln(stdout, versionLine())
		return exitOK
	default:
		fmt.Fprintf(stderr, "%s: unexpected argument %q\n", appName, args[0])
		return exitUsage
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

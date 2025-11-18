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

// version can be overridden at build time.
var version = "0.1.0-dev"

// Exit codes.
const (
	exitOK    = 0
	exitUsage = 2
)

// usage is a raw string literal: everything between the backquotes is taken
// literally, including newlines. No escape sequences are processed.
const usage = `msh — My Parallel SSH

Usage:
  msh [--help] [--version]

Options:
  -h, --help     Show this help and exit.
      --version  Print the version and exit.
`

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
	case "-h", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	case "--version":
		fmt.Fprintln(stdout, versionLine())
		return exitOK
	default:
		fmt.Fprintf(stderr, "%s: unknown argument %q\n", appName, args[0])
		fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", appName)
		return exitUsage
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

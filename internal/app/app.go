// Package app implements the msh command-line interface.
//
// Packages under internal/ can only be imported by code inside this module.
// That lets us change them freely without breaking anyone else.
package app

import (
	"fmt"
	"io"
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

const usage = `msh — My Parallel SSH

Usage:
  msh [--help] [--version]

Options:
  -h, --help     Show this help and exit.
      --version  Print the version and exit.
`

// Banner returns the application's display name. It is exported (starts
// with an upper-case letter) so other packages could use it.
func Banner() string {
	return appName + " — " + appLongName
}

func versionLine() string {
	return fmt.Sprintf("%s version %s", appName, version)
}

// Run executes msh with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, Banner())
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

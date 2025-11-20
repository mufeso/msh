// Package app implements the msh command-line interface.
package app

import (
	"fmt"
	"io"

	"example.com/msh/internal/buildinfo"
)

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

// Run executes msh with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, buildinfo.Banner())
		return exitOK
	}

	switch args[0] {
	case "-h", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	case "--version":
		fmt.Fprintln(stdout, buildinfo.String())
		return exitOK
	default:
		fmt.Fprintf(stderr, "%s: unknown argument %q\n", buildinfo.Name, args[0])
		fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", buildinfo.Name)
		return exitUsage
	}
}

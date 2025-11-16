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

// Exit codes. They become a package of their own in a later step.
const (
	exitOK    = 0
	exitUsage = 2
)

// banner returns the application's display name.
func banner() string {
	return appName + " — " + appLongName
}

// run is the real entry point of the program.
//
// It receives everything it needs as parameters instead of reading global
// state: the command-line arguments (without the program name) and the
// writers for normal output and diagnostics. It returns the process exit
// code instead of calling os.Exit. This makes run easy to test: a test can
// pass a bytes.Buffer and inspect what was written.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintf(stderr, "%s: unexpected argument %q\n", appName, args[0])
		return exitUsage
	}
	fmt.Fprintln(stdout, banner())
	return exitOK
}

func main() {
	// os.Args[0] is the program path; the arguments start at index 1.
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

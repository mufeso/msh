// Package app implements the msh command-line interface.
package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
)

// cli carries everything a command needs. Commands are methods-like
// functions that receive a *cli, so adding a new shared dependency later
// (for example an environment lookup) only means adding a field here.
type cli struct {
	stdout io.Writer
	stderr io.Writer

	// lookupEnv reads environment variables. Run uses os.LookupEnv;
	// tests pass a fake so they never depend on the real environment.
	lookupEnv config.LookupFunc
}

// Run executes msh with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	c := &cli{stdout: stdout, stderr: stderr, lookupEnv: os.LookupEnv}
	return int(c.run(args))
}

func (c *cli) run(args []string) exitcode.Code {
	if len(args) == 0 {
		c.printUsage(c.stdout)
		return exitcode.OK
	}

	switch args[0] {
	case "-h", "--help":
		c.printUsage(c.stdout)
		return exitcode.OK
	case "--version":
		fmt.Fprintln(c.stdout, buildinfo.String())
		return exitcode.OK
	}

	cmd, ok := lookup(args[0])
	if !ok {
		c.errorf("unknown command %q", args[0])
		if s, ok := suggest(args[0], commandNames()); ok {
			fmt.Fprintf(c.stderr, "Did you mean %q?\n", s)
		}
		fmt.Fprintln(c.stderr, "Run 'msh --help' for usage.")
		return exitcode.Usage
	}
	return cmd.run(c, args[1:])
}

// errorf prints a diagnostic prefixed with the program name to stderr.
func (c *cli) errorf(format string, a ...any) {
	fmt.Fprintf(c.stderr, buildinfo.Name+": "+format+"\n", a...)
}

// printUsage writes the top-level help, generated from the command table.
func (c *cli) printUsage(w io.Writer) {
	fmt.Fprintf(w, "%s\n\n", buildinfo.Banner())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  msh <command> [flags]")
	fmt.Fprintln(w, "  msh --help")
	fmt.Fprintln(w, "  msh --version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range commands() {
		fmt.Fprintf(w, "  %-10s %s\n", cmd.name, cmd.summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'msh help <command>' for details about a command.")
}

// ctx returns the context for the current run. Later steps replace it
// with a context that is canceled by CTRL+C.
func (c *cli) ctx() context.Context {
	return context.Background()
}

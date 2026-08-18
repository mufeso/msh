// Package app implements the msh command-line interface.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/signals"
	"example.com/msh/internal/tmux"
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

	// root is canceled by CTRL+C/SIGTERM; every operation derives from it.
	root context.Context

	// term is the local terminal for interactive sessions.
	term interactive.Terminal

	// lookPath finds executables (exec.LookPath); tests replace it.
	lookPath func(file string) (string, error)

	// args is the command line of this run, for the history.
	args []string

	// newTmux returns the tmux runner for a tmux binary path. Tests
	// replace it with an in-memory fake.
	newTmux func(path string) tmux.Runner
}

// Run executes msh with the given arguments (without the program name)
// and returns the process exit code. ctx is the root context: canceling
// it (CTRL+C) stops all work.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	c := &cli{
		stdout: stdout, stderr: stderr, lookupEnv: os.LookupEnv, root: ctx,
		term: interactive.StdTerminal(), lookPath: exec.LookPath,
	}
	return int(c.finish(c.run(args)))
}

// finish makes a run that was interrupted by a signal exit with the
// documented cancellation code, whatever the command returned.
func (c *cli) finish(code exitcode.Code) exitcode.Code {
	if code != exitcode.OK && signals.Interrupted(c.ctx()) {
		return exitcode.Canceled
	}
	return code
}

func (c *cli) run(args []string) exitcode.Code {
	c.args = args
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
		if c.isDestination(args[0]) {
			return runConnect(c, args)
		}
		c.errorf("unknown command %q", args[0])
		if s, ok := suggest(args[0], commandNames()); ok {
			fmt.Fprintf(c.stderr, "Did you mean %q?\n", s)
		}
		fmt.Fprintf(c.stderr, "To connect to a server with this name: msh connect %s\n", args[0])
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
	fmt.Fprintln(w, "  msh [user@]host                like ssh: open a shell")
	fmt.Fprintln(w, "  msh <command> [flags]")
	fmt.Fprintln(w, "  msh --help")
	fmt.Fprintln(w, "  msh --version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range commands() {
		if !cmd.shortcut && !cmd.hidden {
			fmt.Fprintf(w, "  %-13s %s\n", cmd.name, cmd.summary)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Shortcuts (read-only, same selection flags as exec):")
	for _, cmd := range commands() {
		if cmd.shortcut {
			fmt.Fprintf(w, "  %-13s %s\n", cmd.name, cmd.summary)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'msh help <command>' for details about a command.")
}

// ctx returns the root context of this run.
func (c *cli) ctx() context.Context {
	if c.root == nil {
		return context.Background()
	}
	return c.root
}

// terminal returns the terminal for interactive sessions.
func (c *cli) terminal() interactive.Terminal {
	if c.term == nil {
		return interactive.StdTerminal()
	}
	return c.term
}

// tmuxRunner returns the runner for the tmux binary at path.
func (c *cli) tmuxRunner(path string) tmux.Runner {
	if c.newTmux != nil {
		return c.newTmux(path)
	}
	t := c.terminal()
	return tmux.ExecRunner{Path: path, Stdin: t.Stdin(), Stdout: t.Stdout(), Stderr: t.Stderr()}
}

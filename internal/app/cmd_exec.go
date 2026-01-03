package app

import (
	"fmt"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

const execUsage = `Usage: msh exec --local --command CMD [flags]

Runs a command. In this step only --local is supported: the command runs
on this machine with /bin/sh -c. Remote execution over SSH is added later.
`

func execCommand() *command {
	return &command{
		name:    "exec",
		summary: "Run a command on servers",
		usage:   execUsage,
		run:     runExec,
	}
}

func runExec(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("exec", execUsage)
	opts.register(fs)
	command := fs.String("command", "", "command line to run")
	local := fs.Bool("local", false, "run on this machine instead of remote servers")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if *command == "" {
		c.errorf("exec: --command is required")
		return exitcode.Usage
	}
	if !*local {
		c.errorf("exec: remote execution is not available yet; use --local")
		return exitcode.Usage
	}
	if st, code := c.load(&opts, false); st == nil {
		return code
	}

	server := inventory.Server{Name: "local", Address: "localhost"}
	res := executor.LocalExecutor{}.Run(c.ctx(), server, executor.Request{Command: *command})
	fmt.Fprint(c.stdout, res.Stdout)
	fmt.Fprint(c.stderr, res.Stderr)
	switch res.Status {
	case executor.StatusOK:
		return exitcode.OK
	case executor.StatusFailed:
		c.errorf("%s: command exited with code %d", res.Server, res.ExitCode)
		return exitcode.Failure
	default:
		c.errorf("%s: %v", res.Server, res.Err)
		return exitcode.Internal
	}
}

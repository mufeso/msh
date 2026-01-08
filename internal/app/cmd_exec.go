package app

import (
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
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
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}

	server := inventory.Server{Name: "local", Address: "localhost"}
	res := executor.LocalExecutor{}.Run(c.ctx(), server, executor.Request{
		Command: *command,
		Timeout: st.settings.CommandTimeout,
	})
	results := []executor.Result{res}
	if err := output.WriteResults(c.stdout, opts.format, results); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitCodeForResults(results)
}

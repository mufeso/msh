package app

import (
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

const execUsage = `Usage: msh exec (--host NAME | --group NAME | --tag K=V | --all) --command CMD [flags]
       msh exec --local --command CMD [flags]

Runs a command line on a server over SSH, like "ssh user@server CMD".
The command is interpreted by the remote user's shell. It runs on every
selected server.

With --local the command runs on this machine with /bin/sh -c instead.
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
	var sel selectionOptions
	var ro remoteOptions
	fs := c.newFlagSet("exec", execUsage)
	opts.register(fs)
	sel.register(fs)
	ro.register(fs)
	command := fs.String("command", "", "command line to run")
	local := fs.Bool("local", false, "run on this machine instead of remote servers")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if *command == "" {
		c.errorf("exec: --command is required")
		return exitcode.Usage
	}
	if *local {
		if err := sel.validate(); err != nil {
			c.errorf("%v", err)
			return exitcode.Usage
		}
		return c.execLocal(&opts, &ro, *command)
	}
	return c.runRemote(remoteRun{
		opts: &opts, sel: &sel, ro: &ro, name: "exec",
		req: executor.Request{Command: *command},
	})
}

// execLocal runs the command on this machine.
func (c *cli) execLocal(opts *commonOptions, ro *remoteOptions, command string) exitcode.Code {
	st, code := c.load(opts, false)
	if st == nil {
		return code
	}
	server := inventory.Server{Name: "local", Address: "localhost"}
	req := executor.Request{Command: command, Timeout: st.settings.CommandTimeout}
	return c.runAndReport(opts, ro, func(emit func(executor.Result)) runner.Report {
		return runner.Run(c.ctx(), executor.LocalExecutor{}, []inventory.Server{server}, req, runner.Options{}, emit)
	})
}

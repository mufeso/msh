package app

import (
	"context"
	"errors"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/inventory"
)

const shellUsage = `Usage: msh shell --host NAME [--command CMD] [flags]

Opens an interactive login shell on one server, like "ssh user@server",
or runs CMD there with a terminal, like "ssh -t user@server CMD".
msh exits with the remote exit status.

Example:
  msh shell --host server01 --command "podman exec -it asdf /bin/bash"
`

func shellCommand() *command {
	return &command{
		name:    "shell",
		summary: "Open an interactive shell on a server",
		usage:   shellUsage,
		run:     runShell,
	}
}

// shellFlags are the flags of "msh shell".
type shellFlags struct {
	opts commonOptions
	sel  selectionOptions
	ro   remoteOptions
}

func runShell(c *cli, args []string) exitcode.Code {
	var f shellFlags
	fs := c.newFlagSet("shell", shellUsage)
	f.opts.register(fs)
	f.sel.register(fs)
	f.ro.register(fs)
	command := fs.String("command", "", "remote command to run interactively (default: login shell)")
	noTTY := fs.Bool("no-tty", false, "do not request a pseudo-terminal; keep remote stderr separate")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	return c.interactiveSession(&f, interactive.Options{Command: *command, NoTTY: *noTTY})
}

// interactiveSession selects exactly one server, connects and runs an
// interactive session on it.
func (c *cli) interactiveSession(f *shellFlags, iopts interactive.Options) exitcode.Code {
	if err := f.sel.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if f.sel.isEmpty() {
		c.errorf("shell: select a server with --host")
		return exitcode.Usage
	}
	st, code := c.load(&f.opts, true)
	if st == nil {
		return code
	}
	servers, err := st.inv.Select(f.sel.selector())
	if err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if len(servers) != 1 {
		c.errorf("shell: the selection matches %d servers; an interactive session needs exactly one", len(servers))
		return exitcode.Usage
	}
	stack, code := c.newSSH(&f.opts, &f.ro, st)
	if stack == nil {
		return code
	}
	defer stack.Close()
	return c.runInteractive(stack, servers[0], iopts)
}

// runInteractive connects to server and runs the session.
func (c *cli) runInteractive(stack *sshStack, server inventory.Server, iopts interactive.Options) exitcode.Code {
	if iopts.Term == "" {
		iopts.Term = "xterm-256color"
		if v, ok := c.lookupEnv("TERM"); ok && v != "" {
			iopts.Term = v
		}
	}
	client, err := stack.dialer.Connect(c.ctx(), server)
	if err != nil {
		c.errorf("%v", err)
		return errs.ExitCode(errs.KindOf(err))
	}
	defer client.Close()

	status, err := interactive.Run(c.ctx(), client, c.terminal(), iopts)
	if errors.Is(err, context.Canceled) {
		return exitcode.Canceled
	}
	if err != nil {
		c.errorf("%s: %v", server.Name, err)
		return exitcode.Connection
	}
	// Like ssh, pass the remote exit status through.
	return exitcode.Code(status)
}

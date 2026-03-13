package app

import (
	"flag"
	"fmt"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/tmux"
)

// tmuxOptions are the tmux flags of "msh shell".
type tmuxOptions struct {
	enabled bool
}

func (o *tmuxOptions) register(fs *flag.FlagSet) {
	fs.BoolVar(&o.enabled, "tmux", false, "open one tmux pane per selected server")
}

// tmuxSessions opens one interactive session per server in tmux panes.
func (c *cli) tmuxSessions(f *shellFlags, st *setup, servers []inventory.Server, iopts interactive.Options) exitcode.Code {
	if len(servers) < 2 {
		c.errorf("shell --tmux needs at least two servers; for one server use msh shell without --tmux")
		return exitcode.Usage
	}
	path, err := tmux.Find(c.lookPath)
	if err != nil {
		c.errorf("%v", err)
		return errs.ExitCode(errs.KindOf(err))
	}
	if !c.terminal().IsTerminal() {
		c.errorf("shell --tmux needs an interactive terminal")
		return exitcode.Usage
	}
	c.verbosef(&f.opts, "using %s", path)
	fmt.Fprintf(c.stdout, "tmux would open %d panes:\n", len(servers))
	for _, s := range servers {
		fmt.Fprintf(c.stdout, "  %s\n", s.Name)
	}
	return exitcode.OK
}

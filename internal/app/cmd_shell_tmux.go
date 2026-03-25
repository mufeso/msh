package app

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/tmux"
)

// tmuxOptions are the tmux flags of "msh shell".
type tmuxOptions struct {
	enabled  bool
	detached bool
	layout   string // "" = the tmux_layout setting
	session  string // "" = a random msh-xxxxxxxx name
}

func (o *tmuxOptions) register(fs *flag.FlagSet) {
	fs.BoolVar(&o.enabled, "tmux", false, "open one tmux pane per selected server")
	fs.BoolVar(&o.detached, "tmux-detached", false, "create the tmux session but do not attach to it")
	fs.StringVar(&o.session, "tmux-session", "", "tmux session `name` (default: random msh-xxxxxxxx)")
	fs.StringVar(&o.layout, "tmux-layout", "", "tmux `layout`: "+strings.Join(tmux.Layouts, ", ")+" (default: tmux_layout setting)")
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
	if !f.tmux.detached && !c.terminal().IsTerminal() {
		c.errorf("shell --tmux needs an interactive terminal (or use --tmux-detached)")
		return exitcode.Usage
	}
	runner := c.tmuxRunner(path)
	if v, err := tmux.Version(c.ctx(), runner); err == nil {
		c.verbosef(&f.opts, "using %s (%s)", path, v)
	}
	layout := f.tmux.layout
	if layout == "" {
		layout = st.settings.TmuxLayout
	}
	if !tmux.ValidLayout(layout) {
		c.errorf("unknown tmux layout %q (use one of: %s)", layout, strings.Join(tmux.Layouts, ", "))
		return exitcode.Usage
	}
	session := f.tmux.session
	if session == "" {
		session = "msh-" + tmux.RandomToken()
	}
	if !tmux.ValidSessionName(session) {
		c.errorf("invalid --tmux-session %q: use letters, digits, - and _", session)
		return exitcode.Usage
	}
	plan := tmux.Plan{Session: session, Owner: fmt.Sprintf("msh-%d-%s", os.Getpid(), tmux.RandomToken()), Layout: layout}
	for _, s := range servers {
		plan.Panes = append(plan.Panes, tmux.Pane{Title: s.Name, Command: paneCommand(c.paneArgv(f, st, s, iopts))})
	}
	if _, err := tmux.Launch(c.ctx(), runner, plan); err != nil {
		c.errorf("%v", err)
		return errs.ExitCode(errs.KindOf(err))
	}
	fmt.Fprintf(c.stderr, "msh: tmux session %q created\n", session)
	if f.tmux.detached {
		fmt.Fprintf(c.stdout, "%s\n", session)
		return exitcode.OK
	}
	_, inside := c.lookupEnv("TMUX")
	if err := tmux.Attach(c.ctx(), runner, session, inside); err != nil {
		c.errorf("attaching to tmux: %v", err)
		return exitcode.Tmux
	}
	return exitcode.OK
}

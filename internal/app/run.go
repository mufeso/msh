package app

import (
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
)

// remoteRun is everything needed to run one request on selected servers.
type remoteRun struct {
	opts *commonOptions
	sel  *selectionOptions
	ro   *remoteOptions
	req  executor.Request
	name string // command name for messages
}

// runRemote loads settings and inventory, selects servers, builds the SSH
// stack and runs the request. Every remote command (exec, tasks, scripts,
// shortcuts) goes through here, so they all share the same SSH, bastion,
// timeout, cancellation and output behavior.
func (c *cli) runRemote(r remoteRun) exitcode.Code {
	if err := r.sel.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if r.sel.isEmpty() {
		c.errorf("%s: select servers with --host, --group, --tag or --all", r.name)
		return exitcode.Usage
	}
	st, code := c.load(r.opts, true)
	if st == nil {
		return code
	}
	servers, err := st.inv.Select(r.sel.selector())
	if err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if len(servers) != 1 {
		c.errorf("%s: the selection matches %d servers; this version runs on exactly one", r.name, len(servers))
		return exitcode.Usage
	}
	if r.req.Timeout == 0 {
		r.req.Timeout = st.settings.CommandTimeout
	}

	stack, code := c.newSSH(r.opts, r.ro, st)
	if stack == nil {
		return code
	}
	defer stack.Close()

	res := stack.executor.Run(c.ctx(), servers[0], r.req)
	return c.writeResults(r.opts, []executor.Result{res})
}

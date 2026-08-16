package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"example.com/msh/internal/config"
	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/expr"
	"example.com/msh/internal/facts"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/runner"
)

// remoteRun is everything needed to run one request on selected servers.
type remoteRun struct {
	opts *commonOptions
	sel  *selectionOptions
	ro   *remoteOptions
	req  executor.Request
	name string // command name for messages

	unlimited   bool // do not apply command_timeout (e.g. logs --follow)
	quietOutput bool // the output was streamed live; print only statuses
}

// runRemote loads settings and inventory, selects servers, builds the SSH
// stack and runs the request. Every remote command (exec, tasks, scripts,
// shortcuts) goes through here, so they all share the same SSH, bastion,
// timeout, cancellation and output behavior.
func (c *cli) runRemote(r remoteRun) exitcode.Code {
	p, code := c.prepareRun(r.name, r.opts, r.sel, r.ro)
	if p == nil {
		return code
	}
	defer p.stack.Close()
	st, servers, stack := p.st, p.servers, p.stack
	if r.req.Timeout == 0 && !r.unlimited {
		r.req.Timeout = st.settings.CommandTimeout
	}

	ctx, cancel := c.runContext(st)
	defer cancel()
	opts := c.runnerOptions(st, r.ro)
	c.log(r.opts).Debug("run starting", "command", r.name, "servers", len(servers),
		"mode", st.settings.Mode, "workers", opts.Workers, "ordered", opts.Ordered)
	code = c.runAndReport(r.opts, r.ro, func(emit func(executor.Result)) runner.Report {
		if r.quietOutput {
			inner := emit
			emit = func(res executor.Result) {
				res.Stdout, res.Stderr = "", ""
				inner(res)
			}
		}
		return runner.Run(ctx, stack.executor, servers, r.req, opts, emit)
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		c.errorf("global timeout of %s expired", st.settings.Timeout)
	}
	return code
}

// prepared is a run that is ready to start.
type prepared struct {
	st      *setup
	servers []inventory.Server
	stack   *sshStack
}

// selectServers validates the selection, loads settings and the inventory
// and returns the selected servers. On failure it prints the error and
// returns a nil setup and the exit code.
func (c *cli) selectServers(name string, opts *commonOptions, sel *selectionOptions) (*setup, []inventory.Server, exitcode.Code) {
	if err := sel.validate(); err != nil {
		c.errorf("%v", err)
		return nil, nil, exitcode.Usage
	}
	if sel.isEmpty() {
		c.errorf("%s: select servers with --host, --group, --tag or --all", name)
		return nil, nil, exitcode.Usage
	}
	opts.inventoryOptional = sel.onlyHosts()
	st, code := c.load(opts, true)
	if st == nil {
		return nil, nil, code
	}
	if err := c.addTargets(st, sel); err != nil {
		c.errorf("%v", err)
		return nil, nil, exitcode.Usage
	}
	servers, err := st.inv.Select(sel.selector())
	if err != nil {
		c.errorf("%v", err)
		return nil, nil, exitcode.Usage
	}
	return st, servers, exitcode.OK
}

// addTargets turns the --host entries into selectable servers.
func (c *cli) addTargets(st *setup, sel *selectionOptions) error {
	if len(sel.hosts) == 0 {
		return nil
	}
	d := inventory.TargetDefaults{User: st.settings.User, Port: st.settings.Port}
	if d.User == "" {
		d.User = currentUser()
	}
	if b := st.settings.Bastion; b != inventory.NoBastion {
		d.Bastion = b
	}
	source := st.settings.Sources[config.KeyBastion]
	d.ForceBastion = source == config.SourceFlag || source == config.SourceEnv
	names, err := st.inv.AddTargets(sel.hosts, d, nil)
	if err != nil {
		return err
	}
	sel.hosts = names
	return nil
}

// prepareRun selects servers and builds the SSH stack. The caller closes
// the stack.
func (c *cli) prepareRun(name string, opts *commonOptions, sel *selectionOptions, ro *remoteOptions) (*prepared, exitcode.Code) {
	st, servers, code := c.selectServers(name, opts, sel)
	if st == nil {
		return nil, code
	}
	stack, code := c.newSSH(opts, ro, st)
	if stack == nil {
		return nil, code
	}
	servers, code = c.applyWhere(sel, st, stack, servers)
	if servers == nil {
		stack.Close()
		return nil, code
	}
	return &prepared{st: st, servers: servers, stack: stack}, exitcode.OK
}

// applyWhere keeps the servers matching --where. Servers whose facts can't
// be collected are not selected, with a warning. On failure it prints the
// error and returns nil and the exit code.
func (c *cli) applyWhere(sel *selectionOptions, st *setup, stack *sshStack, servers []inventory.Server) ([]inventory.Server, exitcode.Code) {
	if sel.where == "" {
		return servers, exitcode.OK
	}
	e, err := expr.Parse(sel.where)
	if err != nil {
		c.errorf("--where: %v", err)
		return nil, exitcode.Usage
	}
	ctx, cancel := c.runContext(st)
	defer cancel()
	found, failed := c.gatherer(st, stack.executor).Gather(ctx, servers)
	for _, r := range failed {
		fmt.Fprintf(c.stderr, "msh: warning: %s: no facts (%s), not selected\n", r.Server, r.Status)
	}
	var kept []inventory.Server
	for _, s := range servers {
		if f, ok := found[s.Name]; ok && e.Eval(facts.Env(s, f)) {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		c.errorf("%v", inventory.ErrNoMatch)
		return nil, exitcode.Usage
	}
	return kept, exitcode.OK
}

// runContext returns the context for a whole run: the CLI context,
// limited by the global timeout when one is configured.
func (c *cli) runContext(st *setup) (context.Context, context.CancelFunc) {
	if st.settings.Timeout > 0 {
		return context.WithTimeout(c.ctx(), st.settings.Timeout)
	}
	return context.WithCancel(c.ctx())
}

// runAndReport runs work, streaming each result to an output sink as it
// arrives, and returns the exit code the results imply.
func (c *cli) runAndReport(opts *commonOptions, ro *remoteOptions, work func(emit func(executor.Result)) runner.Report) exitcode.Code {
	sink := output.NewSink(c.stdout, opts.format, output.SinkOptions{Prefix: ro.prefix})
	log := c.log(opts)
	var writeErr error
	report := work(func(r executor.Result) {
		// Only metadata is logged: never stdout, stderr or the command.
		log.Debug("result", "server", r.Server, "status", string(r.Status), "exit_code", r.ExitCode,
			"duration_ms", r.Duration.Milliseconds(), "error_kind", errs.KindOf(r.Err).String())
		// Keep the first write error, but keep running: the servers are
		// already working and their results still count.
		if err := sink.Write(r); err != nil && writeErr == nil {
			writeErr = err
		}
	})
	s := report.Summary
	log.Debug("run finished", "total", s.Total, "ok", s.OK, "failed", s.Failed, "error", s.Error,
		"timeout", s.Timeout, "canceled", s.Canceled, "duration_ms", s.Duration.Milliseconds())
	if err := sink.Close(report.Summary); err != nil && writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		c.errorf("writing output: %v", writeErr)
		return exitcode.IO
	}
	if ro.report != "" {
		if err := writeReport(ro.report, report); err != nil {
			c.errorf("%v", errs.New(errs.KindFileWrite, "", err))
			return errs.ExitCode(errs.KindFileWrite)
		}
		log.Debug("report written", "path", ro.report)
	}
	if opts.history && !ro.noHistory {
		c.recordHistory(opts, report)
	}
	return exitCodeForResults(report.Results)
}

// writeReport writes the run as JSON, atomically and readable only by
// the owner (results may contain sensitive output).
func writeReport(path string, report runner.Report) error {
	views := make([]output.ResultView, 0, len(report.Results))
	for _, r := range report.Results { // selection order
		views = append(views, output.NewResultView(r))
	}
	doc := output.NewRunView(views, output.NewSummaryView(report.Summary))
	return output.WriteFileAtomic(path, 0o600, func(w io.Writer) error {
		return output.WriteJSON(w, doc)
	})
}

// runnerOptions turns the settings into runner options. Sequential mode
// is a pool with a single worker.
func (c *cli) runnerOptions(st *setup, ro *remoteOptions) runner.Options {
	workers := st.settings.Workers
	if st.settings.Mode == config.ModeSequential {
		workers = 1
	}
	return runner.Options{Workers: workers, FailFast: ro.failFast, Ordered: st.settings.Ordered}
}

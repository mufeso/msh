package app

import (
	"context"
	"errors"

	"example.com/msh/internal/config"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
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
	if r.req.Timeout == 0 {
		r.req.Timeout = st.settings.CommandTimeout
	}

	stack, code := c.newSSH(r.opts, r.ro, st)
	if stack == nil {
		return code
	}
	defer stack.Close()

	ctx, cancel := c.runContext(st)
	defer cancel()
	code = c.runAndReport(r.opts, r.ro, func(emit func(executor.Result)) runner.Report {
		return runner.Run(ctx, stack.executor, servers, r.req, c.runnerOptions(st, r.ro), emit)
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		c.errorf("global timeout of %s expired", st.settings.Timeout)
	}
	return code
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
	var writeErr error
	report := work(func(r executor.Result) {
		// Keep the first write error, but keep running: the servers are
		// already working and their results still count.
		if err := sink.Write(r); err != nil && writeErr == nil {
			writeErr = err
		}
	})
	if err := sink.Close(report.Summary); err != nil && writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		c.errorf("writing output: %v", writeErr)
		return exitcode.IO
	}
	return exitCodeForResults(report.Results)
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

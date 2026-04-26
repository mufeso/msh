package app

import (
	"fmt"
	"strings"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/runbook"
	"example.com/msh/internal/runner"
)

const runbookRunUsage = `Usage: msh runbook run NAME (--host NAME | --group NAME | --tag K=V | --all) [flags]

Runs the steps of a runbook, in order, on each selected server. With
--dry-run, msh prints what would run without connecting to any server.
`

// runbookRunFlags are the flags of "runbook run".
type runbookRunFlags struct {
	opts   commonOptions
	sel    selectionOptions
	ro     remoteOptions
	dryRun bool
}

func runRunbookRun(c *cli, args []string) exitcode.Code {
	var f runbookRunFlags
	fs := c.newFlagSet("runbook run", runbookRunUsage)
	f.opts.register(fs)
	f.sel.register(fs)
	f.ro.register(fs)
	fs.BoolVar(&f.dryRun, "dry-run", false, "print the plan without connecting to any server")
	names, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(names) != 1 {
		c.errorf("runbook run: exactly one runbook name is required")
		return exitcode.Usage
	}
	st, servers, code := c.selectServers("runbook run", &f.opts, &f.sel)
	if st == nil {
		return code
	}
	rb, code := c.findRunbook(st, names[0])
	if rb == nil {
		return code
	}
	plan, err := rb.Plan(st.tasks, nil)
	if err != nil {
		c.errorf("runbook %q: %v", rb.Name, err)
		return exitcode.Usage
	}
	if f.dryRun {
		return c.printRunbookPlan(&f.opts, rb, servers, plan)
	}
	return c.executeRunbook(&f, st, servers, plan)
}

// executeRunbook runs the plan on the servers. The engine is an Executor,
// so the normal worker pool and output path do the rest.
func (c *cli) executeRunbook(f *runbookRunFlags, st *setup, servers []inventory.Server, plan []runbook.PlannedStep) exitcode.Code {
	stack, code := c.newSSH(&f.opts, &f.ro, st)
	if stack == nil {
		return code
	}
	defer stack.Close()

	engine := &runbook.Engine{Exec: stack.executor, Steps: plan, DefaultTimeout: st.settings.CommandTimeout}
	ctx, cancel := c.runContext(st)
	defer cancel()
	return c.runAndReport(&f.opts, &f.ro, func(emit func(executor.Result)) runner.Report {
		return runner.Run(ctx, engine, servers, executor.Request{}, runner.Options{Workers: 1}, emit)
	})
}

// planView is the JSON shape of a dry run.
type planView struct {
	Runbook string         `json:"runbook"`
	Servers []string       `json:"servers"`
	Steps   []planStepView `json:"steps"`
}

type planStepView struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Stdin   bool   `json:"script_on_stdin"`
}

// printRunbookPlan prints what a run would do.
func (c *cli) printRunbookPlan(opts *commonOptions, rb *runbook.Runbook, servers []inventory.Server, plan []runbook.PlannedStep) exitcode.Code {
	view := planView{Runbook: rb.Name}
	for _, s := range servers {
		view.Servers = append(view.Servers, s.Name)
	}
	for _, ps := range plan {
		view.Steps = append(view.Steps, planStepView{Name: ps.Step.Name, Command: ps.Command, Stdin: ps.Stdin != nil})
	}
	if c.rejectCSV(opts, "runbook run --dry-run") {
		return exitcode.Usage
	}
	if opts.format != output.Text {
		if err := output.WriteObject(c.stdout, opts.format, view); err != nil {
			return exitcode.IO
		}
		return exitcode.OK
	}
	fmt.Fprintf(c.stdout, "Runbook %s: %d steps on %d servers (dry run, nothing was executed)\n", rb.Name, len(plan), len(servers))
	fmt.Fprintf(c.stdout, "Servers: %s\n", strings.Join(view.Servers, ", "))
	for i, s := range view.Steps {
		fmt.Fprintf(c.stdout, "  %d. %s\n     $ %s\n", i+1, s.Name, s.Command)
	}
	return exitcode.OK
}

package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"example.com/msh/internal/config"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/runbook"
	"example.com/msh/internal/runner"
	"example.com/msh/internal/task"
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
	yes    bool
	resume bool
	over   runbook.Overrides
	args   []string
	params map[string]string
}

func runRunbookRun(c *cli, args []string) exitcode.Code {
	var f runbookRunFlags
	fs := c.newFlagSet("runbook run", runbookRunUsage)
	f.opts.register(fs)
	f.sel.register(fs)
	f.ro.register(fs)
	fs.BoolVar(&f.dryRun, "dry-run", false, "print the plan without connecting to any server")
	fs.BoolVar(&f.yes, "yes", false, "confirm running a runbook classified as changes or dangerous")
	fs.BoolVar(&f.resume, "resume", false, "continue a failed run, skipping steps that already finished")
	fs.StringVar(&f.over.Batch, "batch", "", "servers per batch: a `number` or percentage, e.g. 2 or 25%")
	fs.StringVar(&f.over.MaxFailures, "max-failures", "", "failed servers tolerated before stopping: a `number` or percentage")
	f.params = map[string]string{}
	fs.Func("arg", "runbook argument `value`, in order (repeatable)", func(v string) error {
		f.args = append(f.args, v)
		return nil
	})
	fs.Func("param", "named runbook argument `name=value` (repeatable)", func(v string) error {
		name, value, ok := strings.Cut(v, "=")
		if !ok || name == "" {
			return fmt.Errorf("%q is not in name=value form", v)
		}
		f.params[name] = value
		return nil
	})
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
	rb, err := rb.WithOverrides(f.over)
	if err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	values, err := rb.Bind(f.args, f.params)
	if err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	plan, err := rb.Plan(st.tasks, values)
	if err != nil {
		c.errorf("runbook %q: %v", rb.Name, err)
		return exitcode.Usage
	}
	safety := rb.Safety(st.tasks)
	if f.dryRun {
		return c.printRunbookPlan(&f.opts, rb, safety, servers, plan)
	}
	if safety != task.SafetyReadOnly && !f.yes {
		c.errorf("runbook %q is classified %q; review it with --dry-run, then add --yes to run it", rb.Name, safety)
		return exitcode.Usage
	}
	return c.executeRunbook(&f, st, rb, servers, plan)
}

// executeRunbook runs the plan on the servers. The engine is an Executor,
// so the normal worker pool and output path do the rest.
func (c *cli) executeRunbook(f *runbookRunFlags, st *setup, rb *runbook.Runbook, servers []inventory.Server, plan []runbook.PlannedStep) exitcode.Code {
	stack, code := c.newSSH(&f.opts, &f.ro, st)
	if stack == nil {
		return code
	}
	defer stack.Close()

	engine := &runbook.Engine{Exec: stack.executor, Steps: plan, DefaultTimeout: st.settings.CommandTimeout}
	if rb.Timeout != nil {
		engine.ServerTimeout = time.Duration(*rb.Timeout)
	}
	tracker, code := c.runbookTracker(f, rb, plan)
	if tracker == nil {
		return code
	}
	engine.Progress = tracker

	rollout, err := rb.NewRollout(len(servers))
	if err != nil {
		c.errorf("runbook %q: %v", rb.Name, err)
		return exitcode.Usage
	}

	ctx, cancel := c.runContext(st)
	defer cancel()
	var succeeded bool
	code = c.runAndReport(&f.opts, &f.ro, func(emit func(executor.Result)) runner.Report {
		report := rollout.Run(ctx, engine, servers, c.runnerOptions(st, &f.ro), emit)
		succeeded = report.Summary.Succeeded()
		return report
	})
	if err := tracker.Err(); err != nil {
		c.errorf("saving runbook progress: %v", err)
	}
	if succeeded {
		if err := tracker.Finish(); err != nil {
			c.errorf("removing runbook progress: %v", err)
		}
	} else {
		fmt.Fprintf(c.stderr, "msh: to continue after fixing the problem, run it again with --resume\n")
	}
	return code
}

// runbookTracker opens the progress file of a run. On failure it prints
// the error and returns nil and the exit code.
func (c *cli) runbookTracker(f *runbookRunFlags, rb *runbook.Runbook, plan []runbook.PlannedStep) (*runbook.Tracker, exitcode.Code) {
	dir := config.StateDir(c.lookupEnv)
	if dir == "" {
		c.errorf("cannot find a state directory: set HOME or XDG_STATE_HOME")
		return nil, exitcode.Config
	}
	path := filepath.Join(dir, "runbooks", rb.Name+".json")
	tracker, err := runbook.NewTracker(path, rb.Name, runbook.PlanHash(plan), f.resume)
	if err != nil {
		c.errorf("runbook %q: %v", rb.Name, err)
		return nil, exitcode.Usage
	}
	return tracker, exitcode.OK
}

// planView is the JSON shape of a dry run.
type planView struct {
	Runbook string         `json:"runbook"`
	Safety  string         `json:"safety"`
	Servers []string       `json:"servers"`
	Steps   []planStepView `json:"steps"`
}

type planStepView struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Stdin   bool   `json:"script_on_stdin"`
}

// printRunbookPlan prints what a run would do.
func (c *cli) printRunbookPlan(opts *commonOptions, rb *runbook.Runbook, safety task.Safety, servers []inventory.Server, plan []runbook.PlannedStep) exitcode.Code {
	view := planView{Runbook: rb.Name, Safety: string(safety)}
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
	fmt.Fprintf(c.stdout, "Safety:  %s\n", safety)
	fmt.Fprintf(c.stdout, "Servers: %s\n", strings.Join(view.Servers, ", "))
	for i, s := range view.Steps {
		fmt.Fprintf(c.stdout, "  %d. %s\n     $ %s\n", i+1, s.Name, s.Command)
	}
	return exitcode.OK
}

package app

import (
	"flag"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/task"
)

const taskRunUsage = `Usage: msh task run NAME (--host NAME | --group NAME | --tag K=V | --all) [flags]

Runs a configured task on the selected servers. Tasks classified as
"changes" or "dangerous" also require --yes.
`

const cUsage = `Usage: msh c TASK SERVER [flags]

Short form of: msh task run TASK --host SERVER
`

// taskRunFlags are the flags of "task run" and "c".
type taskRunFlags struct {
	opts commonOptions
	sel  selectionOptions
	ro   remoteOptions
	yes  bool
}

func (c *cli) newTaskRunFlags(name, usage string) (*taskRunFlags, *flag.FlagSet) {
	f := &taskRunFlags{}
	fs := c.newFlagSet(name, usage)
	f.opts.register(fs)
	f.sel.register(fs)
	f.ro.register(fs)
	fs.BoolVar(&f.yes, "yes", false, "confirm running a task classified as changes or dangerous")
	return f, fs
}

func runTaskRun(c *cli, args []string) exitcode.Code {
	f, fs := c.newTaskRunFlags("task run", taskRunUsage)
	positional, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(positional) != 1 {
		c.errorf("task run: exactly one task name is required")
		return exitcode.Usage
	}
	return c.runTaskOn(f, positional[0])
}

// cCommand is the short alias: msh c TASK SERVER.
func cCommand() *command {
	return &command{
		name:    "c",
		summary: "Short form of task run: msh c TASK SERVER",
		usage:   cUsage,
		run:     runC,
	}
}

func runC(c *cli, args []string) exitcode.Code {
	f, fs := c.newTaskRunFlags("c", cUsage)
	positional, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(positional) < 2 {
		c.errorf("c: usage: msh c TASK SERVER")
		return exitcode.Usage
	}
	if len(positional) > 2 {
		c.errorf("c: too many arguments")
		return exitcode.Usage
	}
	f.sel.hosts = append(f.sel.hosts, positional[1])
	return c.runTaskOn(f, positional[0])
}

// runTaskOn renders the task and runs it on the selected servers.
func (c *cli) runTaskOn(f *taskRunFlags, name string) exitcode.Code {
	st, code := c.load(&f.opts, false)
	if st == nil {
		return code
	}
	t, code := c.findTask(st, name)
	if t == nil {
		return code
	}
	if s := t.EffectiveSafety(); s != task.SafetyReadOnly && !f.yes {
		c.errorf("task %q is classified %q; add --yes to run it", t.Name, s)
		return exitcode.Usage
	}
	inv, err := t.Render()
	if err != nil {
		c.errorf("task %q: %v", t.Name, err)
		return exitcode.Config
	}
	if inv.Interactive {
		c.errorf("task %q is interactive; interactive sessions are not available yet", t.Name)
		return exitcode.Usage
	}
	return c.runRemote(remoteRun{
		opts: &f.opts, sel: &f.sel, ro: &f.ro, name: "task " + t.Name,
		req: executor.Request{Command: inv.Command, Stdin: inv.Stdin, Timeout: inv.Timeout},
	})
}

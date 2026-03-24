package app

import (
	"flag"
	"fmt"
	"strings"

	"example.com/msh/internal/cliflag"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/task"
)

const taskRunUsage = `Usage: msh task run NAME (--host NAME | --group NAME | --tag K=V | --all) [--arg VALUE]... [--param NAME=VALUE]... [flags]

Runs a configured task on the selected servers. Tasks classified as
"changes" or "dangerous" also require --yes.
`

const cUsage = `Usage: msh c TASK SERVER [ARG]... [flags]

Short form of: msh task run TASK --host SERVER --arg ARG...
`

// taskRunFlags are the flags of "task run" and "c".
type taskRunFlags struct {
	opts   commonOptions
	sel    selectionOptions
	ro     remoteOptions
	tmux   tmuxOptions
	yes    bool
	args   cliflag.StringList
	params map[string]string
}

func (c *cli) newTaskRunFlags(name, usage string) (*taskRunFlags, *flag.FlagSet) {
	f := &taskRunFlags{}
	fs := c.newFlagSet(name, usage)
	f.opts.register(fs)
	f.sel.register(fs)
	f.ro.register(fs)
	f.tmux.register(fs)
	fs.BoolVar(&f.yes, "yes", false, "confirm running a task classified as changes or dangerous")
	fs.Func("arg", "task argument `value`, in order (repeatable)", func(v string) error {
		// fs.Func instead of cliflag.StringList: argument values may
		// legitimately contain commas.
		f.args = append(f.args, v)
		return nil
	})
	f.params = map[string]string{}
	fs.Func("param", "named task argument `name=value` (repeatable)", func(v string) error {
		name, value, ok := strings.Cut(v, "=")
		if !ok || name == "" {
			return fmt.Errorf("%q is not in name=value form", v)
		}
		if _, dup := f.params[name]; dup {
			return fmt.Errorf("argument %q given twice", name)
		}
		f.params[name] = value
		return nil
	})
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
	f.sel.hosts = append(f.sel.hosts, positional[1])
	f.args = append(f.args, positional[2:]...)
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
	values, err := t.Bind(f.args, f.params)
	if err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	inv, err := t.Render(values)
	if err != nil {
		c.errorf("task %q: %v", t.Name, err)
		return exitcode.Config
	}
	if inv.Interactive {
		sf := shellFlags{opts: f.opts, sel: f.sel, ro: f.ro, tmux: f.tmux}
		return c.interactiveSession(&sf, interactive.Options{Command: inv.Command})
	}
	return c.runRemote(remoteRun{
		opts: &f.opts, sel: &f.sel, ro: &f.ro, name: "task " + t.Name,
		req: executor.Request{Command: inv.Command, Stdin: inv.Stdin, Timeout: inv.Timeout},
	})
}

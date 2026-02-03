package app

import (
	"fmt"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/task"
)

const taskShowUsage = `Usage: msh task show NAME [flags]

Shows one task: how it executes, its arguments and its template or script.
`

func runTaskShow(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("task show", taskShowUsage)
	opts.register(fs)
	names, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(names) != 1 {
		c.errorf("task show: exactly one task name is required")
		return exitcode.Usage
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}
	t, code := c.findTask(st, names[0])
	if t == nil {
		return code
	}
	if opts.format == output.JSON {
		if err := output.WriteJSON(c.stdout, t); err != nil {
			return exitcode.IO
		}
		return exitcode.OK
	}
	fmt.Fprint(c.stdout, describeTask(t))
	return exitcode.OK
}

// findTask looks up a task by name or alias and prints a suggestion when
// it does not exist.
func (c *cli) findTask(st *setup, name string) (*task.Task, exitcode.Code) {
	t, ok := st.tasks.Get(name)
	if ok {
		return t, exitcode.OK
	}
	c.errorf("unknown task %q", name)
	if s, ok := suggest(name, st.tasks.Names()); ok {
		fmt.Fprintf(c.stderr, "Did you mean %q?\n", s)
	}
	return nil, exitcode.Usage
}

// describeTask renders a task for humans.
func describeTask(t *task.Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name:        %s\n", t.Name)
	if len(t.Aliases) > 0 {
		fmt.Fprintf(&b, "Aliases:     %s\n", strings.Join(t.Aliases, ", "))
	}
	if t.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", t.Description)
	}
	fmt.Fprintf(&b, "Mode:        %s\n", modeDescription(t))
	fmt.Fprintf(&b, "Safety:      %s\n", t.EffectiveSafety())
	fmt.Fprintf(&b, "Interactive: %v\n", t.Interactive)
	if t.Timeout != nil {
		fmt.Fprintf(&b, "Timeout:     %s\n", t.Timeout)
	}
	if len(t.Arguments) > 0 {
		b.WriteString("Arguments:\n")
		for i, a := range t.Arguments {
			fmt.Fprintf(&b, "  %d. %s", i+1, a.Name)
			switch {
			case a.Required:
				b.WriteString(" (required)")
			case a.Default != nil:
				fmt.Fprintf(&b, " (default %q)", *a.Default)
			default:
				b.WriteString(" (optional)")
			}
			if a.Description != "" {
				fmt.Fprintf(&b, " - %s", a.Description)
			}
			b.WriteString("\n")
		}
	}
	switch t.Mode() {
	case task.ModeDirect:
		fmt.Fprintf(&b, "Argv:        %q\n", t.Argv)
	case task.ModeShell:
		fmt.Fprintf(&b, "Command:     %s\n", t.Command)
	case task.ModeScript:
		b.WriteString("Script:\n")
		for _, line := range strings.Split(strings.TrimRight(t.Script, "\n"), "\n") {
			fmt.Fprintf(&b, "  | %s\n", line)
		}
	}
	return b.String()
}

// modeDescription explains how the task executes, so shell-interpreted
// tasks are always clearly marked.
func modeDescription(t *task.Task) string {
	interp := t.Interpreter
	if interp == "" {
		interp = "sh"
	}
	switch t.Mode() {
	case task.ModeDirect:
		return "direct (no shell; arguments are passed as separate words)"
	case task.ModeScript:
		return fmt.Sprintf("script streamed to %s on standard input; arguments are positional parameters", interp)
	default:
		return fmt.Sprintf("SHELL (%s -c); argument values are inserted as quoted words", interp)
	}
}

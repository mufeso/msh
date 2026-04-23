package app

import (
	"fmt"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/runbook"
)

const runbookShowUsage = `Usage: msh runbook show NAME [flags]

Shows the steps of one runbook.
`

func runRunbookShow(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("runbook show", runbookShowUsage)
	opts.register(fs)
	names, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(names) != 1 {
		c.errorf("runbook show: exactly one runbook name is required")
		return exitcode.Usage
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}
	rb, code := c.findRunbook(st, names[0])
	if rb == nil {
		return code
	}
	if c.rejectCSV(&opts, "runbook show") {
		return exitcode.Usage
	}
	if opts.format != output.Text {
		if err := output.WriteObject(c.stdout, opts.format, rb); err != nil {
			return exitcode.IO
		}
		return exitcode.OK
	}
	fmt.Fprint(c.stdout, describeRunbook(rb))
	return exitcode.OK
}

// findRunbook looks up a runbook and suggests a close name on failure.
func (c *cli) findRunbook(st *setup, name string) (*runbook.Runbook, exitcode.Code) {
	rb, ok := st.runbooks.Get(name)
	if ok {
		return rb, exitcode.OK
	}
	c.errorf("unknown runbook %q", name)
	if s, ok := suggest(name, st.runbooks.Names()); ok {
		fmt.Fprintf(c.stderr, "Did you mean %q?\n", s)
	}
	return nil, exitcode.Usage
}

// describeRunbook renders a runbook for humans.
func describeRunbook(rb *runbook.Runbook) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name:        %s\n", rb.Name)
	if rb.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", rb.Description)
	}
	b.WriteString("Steps:\n")
	for i, s := range rb.Steps {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, s.Name)
		if s.Task != "" {
			fmt.Fprintf(&b, "     task: %s\n", s.Task)
		} else {
			fmt.Fprintf(&b, "     command: %s\n", s.Command)
		}
		if s.Retries > 0 {
			fmt.Fprintf(&b, "     retries: %d\n", s.Retries)
		}
		if s.Timeout != nil {
			fmt.Fprintf(&b, "     timeout: %s\n", s.Timeout)
		}
		if s.ContinueOnError {
			b.WriteString("     continues on error\n")
		}
	}
	return b.String()
}

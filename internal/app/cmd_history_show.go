package app

import (
	"errors"
	"fmt"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/history"
	"example.com/msh/internal/output"
)

const historyShowUsage = `Usage: msh history show REF [flags]

Shows a recorded run. REF is "last", an ID, or a unique part of the start
of an ID.
`

func runHistoryShow(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("history show", historyShowUsage)
	opts.register(fs)
	refs, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(refs) != 1 {
		c.errorf("history show: exactly one run is required")
		return exitcode.Usage
	}
	if st, code := c.load(&opts, false); st == nil {
		return code
	}
	if c.rejectCSV(&opts, "history show") {
		return exitcode.Usage
	}
	r, code := c.findRun(refs[0])
	if code != exitcode.OK {
		return code
	}
	if opts.format != output.Text {
		if err := output.WriteObject(c.stdout, opts.format, r); err != nil {
			return exitcode.IO
		}
		return exitcode.OK
	}
	fmt.Fprint(c.stdout, describeRun(r))
	return exitcode.OK
}

// findRun looks up a run and prints an error if it can't.
func (c *cli) findRun(ref string) (history.Record, exitcode.Code) {
	store, ok := c.historyStore()
	if !ok {
		c.errorf("cannot find a state directory: set HOME or XDG_STATE_HOME")
		return history.Record{}, exitcode.Config
	}
	r, err := store.Find(ref)
	if errors.Is(err, history.ErrNotFound) {
		c.errorf("%v: %q", err, ref)
		return r, exitcode.Usage
	}
	if err != nil {
		c.errorf("%v", err)
		return r, exitcode.Usage
	}
	return r, exitcode.OK
}

// describeRun renders a record for people.
func describeRun(r history.Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Run:     %s\nStarted: %s\nCommand: %s\n", r.ID, r.Started.Local().Format("2006-01-02 15:04:05"), r.Command)
	fmt.Fprintf(&b, "Result:  %d servers, %d ok, %d with problems, %d ms\n", r.Summary.Total, r.Summary.OK, r.Summary.Total-r.Summary.OK, r.Summary.DurationMS)
	for _, v := range r.Results {
		fmt.Fprintf(&b, "=== %s: %s (exit %d, %d ms)\n", v.Server, v.Status, v.ExitCode, v.DurationMS)
		if v.Error != "" {
			fmt.Fprintf(&b, "error: %s\n", v.Error)
		}
		b.WriteString(v.Stdout)
		if v.Stderr != "" {
			b.WriteString("--- stderr\n")
			b.WriteString(v.Stderr)
		}
	}
	return b.String()
}

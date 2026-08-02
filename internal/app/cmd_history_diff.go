package app

import (
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/history"
	"example.com/msh/internal/output"
)

const historyDiffUsage = `Usage: msh history diff REF1 REF2 [flags]

Compares two recorded runs server by server. Exits 1 if they differ.
`

func runHistoryDiff(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("history diff", historyDiffUsage)
	opts.register(fs)
	refs, code, ok := parseInterspersed(fs, args)
	if !ok {
		return code
	}
	if len(refs) != 2 {
		c.errorf("history diff: exactly two runs are required")
		return exitcode.Usage
	}
	if st, code := c.load(&opts, false); st == nil {
		return code
	}
	a, code := c.findRun(refs[0])
	if code != exitcode.OK {
		return code
	}
	b, code := c.findRun(refs[1])
	if code != exitcode.OK {
		return code
	}
	changes := history.Diff(a, b)
	rows := make([][]string, 0, len(changes))
	for _, ch := range changes {
		rows = append(rows, []string{ch.Server, ch.Field, ch.Before, ch.After})
	}
	if err := output.WriteRows(c.stdout, opts.format, []string{"SERVER", "FIELD", "BEFORE", "AFTER"}, rows, changes); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	if len(changes) > 0 {
		return exitcode.Failure
	}
	return exitcode.OK
}

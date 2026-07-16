package app

import (
	"fmt"
	"strconv"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const historyUsage = `Usage: msh history <subcommand> [flags]

Subcommands:
  list    List recorded runs
  show    Show one recorded run

msh records runs in ~/.local/state/msh/history. See docs/history.md.
`

const historyListUsage = `Usage: msh history list [--limit N] [flags]

Lists recorded runs, newest first.
`

func historyCommand() *command {
	return &command{
		name:    "history",
		summary: "List, show and compare recorded runs",
		usage:   historyUsage,
		run:     runHistory,
	}
}

func runHistory(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, historyUsage)
		return exitcode.Usage
	}
	switch args[0] {
	case "list":
		return runHistoryList(c, args[1:])
	case "show":
		return runHistoryShow(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, historyUsage)
		return exitcode.OK
	default:
		c.errorf("unknown history subcommand %q", args[0])
		fmt.Fprint(c.stderr, historyUsage)
		return exitcode.Usage
	}
}

// historyRow is one run in "history list".
type historyRow struct {
	ID       string `json:"id"`
	Started  string `json:"started"`
	Servers  int    `json:"servers"`
	OK       int    `json:"ok"`
	Problems int    `json:"problems"`
	Command  string `json:"command"`
}

func runHistoryList(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("history list", historyListUsage)
	opts.register(fs)
	limit := fs.Int("limit", 20, "show at most this many `runs` (0 = all)")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if st, code := c.load(&opts, false); st == nil {
		return code
	}
	store, ok := c.historyStore()
	if !ok {
		c.errorf("cannot find a state directory: set HOME or XDG_STATE_HOME")
		return exitcode.Config
	}
	items := []historyRow{}
	rows := [][]string{}
	for r, err := range store.All() {
		if err != nil {
			fmt.Fprintf(c.stderr, "msh: warning: %v\n", err)
			continue
		}
		s := r.Summary
		row := historyRow{
			ID: r.ID, Started: r.Started.Local().Format("2006-01-02 15:04:05"),
			Servers: s.Total, OK: s.OK, Problems: s.Total - s.OK, Command: r.Command,
		}
		items = append(items, row)
		rows = append(rows, []string{row.ID, row.Started, strconv.Itoa(row.Servers), strconv.Itoa(row.OK), strconv.Itoa(row.Problems), row.Command})
		if *limit > 0 && len(items) >= *limit {
			break // the iterator stops reading files here
		}
	}
	if err := output.WriteRows(c.stdout, opts.format, []string{"ID", "STARTED", "SERVERS", "OK", "PROBLEMS", "COMMAND"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

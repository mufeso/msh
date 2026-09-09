package app

import (
	"fmt"
	"strconv"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/history"
	"example.com/msh/internal/output"
)

const historyUsage = `Usage: msh history <subcommand> [flags]

Subcommands:
  list    List recorded runs
  show    Show one recorded run
  diff    Compare two recorded runs
  prune   Remove old runs

msh records runs in ~/.local/state/msh/history. See docs/history.md.
`

const historyListUsage = `Usage: msh history list [PATTERN] [--limit N] [flags]

Lists recorded runs, newest first. PATTERN works like grep on the command
line, and can also be given with --grep.
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
	case "diff":
		return runHistoryDiff(c, args[1:])
	case "prune":
		return runHistoryPrune(c, args[1:])
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
	failed := fs.Bool("failed", false, "only runs with problems")
	server := fs.String("server", "", "only runs that included this `server`")
	since := fs.Duration("since", 0, "only runs from this recent `time`, e.g. 24h")
	pattern := fs.String("grep", "", "only runs whose command line contains this `pattern`")
	var grep grepFlags
	grep.register(fs)
	match, code, ok := grep.parseList(c, "history list", fs, args, pattern)
	if !ok {
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
	oldest := time.Now().Add(-*since)
	matching := history.Filter(store.All(), func(r history.Record) bool {
		return (!*failed || r.Summary.Total != r.Summary.OK) &&
			(*server == "" || r.HasServer(*server)) &&
			(*since == 0 || r.Started.After(oldest)) &&
			match.Match(r.Command)
	})
	for r, err := range matching {
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

const historyPruneUsage = `Usage: msh history prune --keep N

Removes all but the newest N recorded runs.
`

func runHistoryPrune(c *cli, args []string) exitcode.Code {
	fs := c.newFlagSet("history prune", historyPruneUsage)
	keep := fs.Int("keep", -1, "keep this many `runs`")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if *keep < 0 {
		c.errorf("history prune: --keep N is required")
		return exitcode.Usage
	}
	store, ok := c.historyStore()
	if !ok {
		c.errorf("cannot find a state directory: set HOME or XDG_STATE_HOME")
		return exitcode.Config
	}
	removed, err := store.Prune(*keep)
	if err != nil {
		c.errorf("%v", err)
		return exitcode.IO
	}
	fmt.Fprintf(c.stdout, "removed %d runs\n", removed)
	return exitcode.OK
}

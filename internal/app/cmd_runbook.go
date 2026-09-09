package app

import (
	"fmt"
	"strconv"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const runbookUsage = `Usage: msh runbook <subcommand> [flags]

Subcommands:
  list    List configured runbooks
  show    Show the steps of one runbook
  run     Run a runbook on servers

Runbooks are defined under "runbooks:" in the configuration file.
`

const runbookListUsage = `Usage: msh runbook list [PATTERN] [flags]

Lists the runbooks from the configuration file. PATTERN works like grep on
each row.
`

func runbookCommand() *command {
	return &command{
		name:    "runbook",
		summary: "List, show and run multi-step runbooks",
		usage:   runbookUsage,
		run:     runRunbook,
	}
}

func runRunbook(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, runbookUsage)
		return exitcode.Usage
	}
	switch args[0] {
	case "list":
		return runRunbookList(c, args[1:])
	case "show":
		return runRunbookShow(c, args[1:])
	case "run":
		return runRunbookRun(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, runbookUsage)
		return exitcode.OK
	default:
		c.errorf("unknown runbook subcommand %q", args[0])
		fmt.Fprint(c.stderr, runbookUsage)
		return exitcode.Usage
	}
}

// runbookSummary is the JSON shape of one runbook in "runbook list".
type runbookSummary struct {
	Name        string `json:"name"`
	Steps       int    `json:"steps"`
	Description string `json:"description"`
}

func runRunbookList(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("runbook list", runbookListUsage)
	opts.register(fs)
	var grep grepFlags
	grep.register(fs)
	match, code, ok := grep.parseList(c, "runbook list", fs, args, nil)
	if !ok {
		return code
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}
	books := st.runbooks.List()
	summaries := make([]runbookSummary, 0, len(books))
	rows := make([][]string, 0, len(books))
	for _, rb := range books {
		s := runbookSummary{Name: rb.Name, Steps: len(rb.Steps), Description: rb.Description}
		row := []string{s.Name, strconv.Itoa(s.Steps), s.Description}
		if !match.Match(strings.Join(row, " ")) {
			continue
		}
		summaries = append(summaries, s)
		rows = append(rows, row)
	}
	if err := output.WriteRows(c.stdout, opts.format, []string{"NAME", "STEPS", "DESCRIPTION"}, rows, summaries); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

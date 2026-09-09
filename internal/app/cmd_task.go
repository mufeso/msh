package app

import (
	"fmt"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/task"
)

const taskUsage = `Usage: msh task <subcommand> [flags]

Subcommands:
  list    List configured tasks
  show    Show one task in detail
  run     Run a task on servers

Tasks are defined under "tasks:" in the configuration file.
`

const taskListUsage = `Usage: msh task list [PATTERN] [flags]

Lists the tasks from the configuration file. PATTERN works like grep on each
row.
`

func taskCommand() *command {
	return &command{
		name:    "task",
		summary: "List, show and run configured tasks",
		usage:   taskUsage,
		run:     runTask,
	}
}

func runTask(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, taskUsage)
		return exitcode.Usage
	}
	switch args[0] {
	case "list":
		return runTaskList(c, args[1:])
	case "show":
		return runTaskShow(c, args[1:])
	case "run":
		return runTaskRun(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, taskUsage)
		return exitcode.OK
	default:
		c.errorf("unknown task subcommand %q", args[0])
		fmt.Fprint(c.stderr, taskUsage)
		return exitcode.Usage
	}
}

// taskSummary is the JSON shape of one task in "task list".
type taskSummary struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Mode        string   `json:"mode"`
	Safety      string   `json:"safety"`
	Interactive bool     `json:"interactive"`
	Description string   `json:"description"`
}

func summarize(t *task.Task) taskSummary {
	aliases := t.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	return taskSummary{
		Name: t.Name, Aliases: aliases, Mode: string(t.Mode()),
		Safety: string(t.EffectiveSafety()), Interactive: t.Interactive, Description: t.Description,
	}
}

func runTaskList(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("task list", taskListUsage)
	opts.register(fs)
	var grep grepFlags
	grep.register(fs)
	match, code, ok := grep.parseList(c, "task list", fs, args, nil)
	if !ok {
		return code
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}

	tasks := st.tasks.List()
	summaries := make([]taskSummary, 0, len(tasks))
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		s := summarize(t)
		row := []string{s.Name, strings.Join(s.Aliases, ","), s.Mode, s.Safety, s.Description}
		if !match.Match(strings.Join(row, " ")) {
			continue
		}
		summaries = append(summaries, s)
		rows = append(rows, row)
	}
	headers := []string{"NAME", "ALIASES", "MODE", "SAFETY", "DESCRIPTION"}
	if err := output.WriteRows(c.stdout, opts.format, headers, rows, summaries); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

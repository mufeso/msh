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

Tasks are defined under "tasks:" in the configuration file.
`

const taskListUsage = `Usage: msh task list [flags]

Lists the tasks from the configuration file.
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
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}

	tasks := st.tasks.List()
	var err error
	if opts.format == output.JSON {
		summaries := make([]taskSummary, 0, len(tasks))
		for _, t := range tasks {
			summaries = append(summaries, summarize(t))
		}
		err = output.WriteJSON(c.stdout, summaries)
	} else {
		rows := make([][]string, 0, len(tasks))
		for _, t := range tasks {
			s := summarize(t)
			rows = append(rows, []string{s.Name, strings.Join(s.Aliases, ","), s.Mode, s.Safety, s.Description})
		}
		err = output.Table(c.stdout, []string{"NAME", "ALIASES", "MODE", "SAFETY", "DESCRIPTION"}, rows)
	}
	if err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

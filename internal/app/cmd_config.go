package app

import (
	"fmt"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const configUsage = `Usage: msh config <subcommand> [flags]

Subcommands:
  show      Show effective settings and where each value came from

Precedence: flag > environment variable > configuration file > default.
`

const configShowUsage = `Usage: msh config show [flags]

Shows every effective setting, its value and its source.
`

func configCommand() *command {
	return &command{
		name:    "config",
		summary: "Show and validate configuration",
		usage:   configUsage,
		run:     runConfig,
	}
}

func runConfig(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, configUsage)
		return exitcode.Usage
	}
	switch args[0] {
	case "show":
		return runConfigShow(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, configUsage)
		return exitcode.OK
	default:
		c.errorf("unknown config subcommand %q", args[0])
		fmt.Fprint(c.stderr, configUsage)
		return exitcode.Usage
	}
}

// settingView is the JSON shape of one setting.
type settingView struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

func runConfigShow(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("config show", configShowUsage)
	opts.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	s, code, ok := c.loadSettings(&opts)
	if !ok {
		return code
	}

	views := make([]settingView, 0, len(config.Keys()))
	for _, k := range config.Keys() {
		views = append(views, settingView{Key: k, Value: s.Value(k), Source: s.Sources[k]})
	}

	var err error
	if opts.format == output.JSON {
		err = output.WriteJSON(c.stdout, views)
	} else {
		rows := make([][]string, 0, len(views))
		for _, v := range views {
			value := v.Value
			if value == "" {
				value = "-"
			}
			rows = append(rows, []string{v.Key, value, v.Source})
		}
		err = output.Table(c.stdout, []string{"KEY", "VALUE", "SOURCE"}, rows)
	}
	if err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

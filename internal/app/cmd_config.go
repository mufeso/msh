package app

import (
	"fmt"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const configUsage = `Usage: msh config <subcommand> [flags]

Subcommands:
  init      Write a starter configuration and inventory
  show      Show effective settings and where each value came from
  validate  Validate the configuration and the inventory

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
	case "init":
		return runConfigInit(c, args[1:])
	case "show":
		return runConfigShow(c, args[1:])
	case "validate":
		return runConfigValidate(c, args[1:])
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
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}
	s := st.settings

	views := make([]settingView, 0, len(config.Keys()))
	for _, k := range config.Keys() {
		views = append(views, settingView{Key: k, Value: s.Value(k), Source: s.Sources[k]})
	}

	rows := make([][]string, 0, len(views))
	for _, v := range views {
		value := v.Value
		if value == "" && opts.format == output.Text {
			value = "-"
		}
		rows = append(rows, []string{v.Key, value, v.Source})
	}
	if err := output.WriteRows(c.stdout, opts.format, []string{"KEY", "VALUE", "SOURCE"}, rows, views); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

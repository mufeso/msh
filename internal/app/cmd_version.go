package app

import (
	"fmt"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const versionUsage = `Usage: msh version [--short] [--output text|json]

Prints the msh version, source commit, build date, Go version and platform.
`

func versionCommand() *command {
	return &command{
		name:    "version",
		summary: "Print version information",
		usage:   versionUsage,
		run:     runVersion,
	}
}

func runVersion(c *cli, args []string) exitcode.Code {
	fs := c.newFlagSet("version", versionUsage)
	short := fs.Bool("short", false, "print only the version number")
	outputName := fs.String("output", "text", "output `format`: text or json")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() > 0 {
		c.errorf("version takes no arguments")
		return exitcode.Usage
	}
	format, err := output.ParseFormat(*outputName)
	if err != nil {
		c.errorf("--output: %v", err)
		return exitcode.Usage
	}

	info := buildinfo.Get()
	switch {
	case *short:
		fmt.Fprintln(c.stdout, info.Version)
	case format == output.JSON:
		err = output.WriteJSON(c.stdout, info)
	default:
		err = output.KeyValues(c.stdout, []output.Pair{
			{Key: "Name", Value: info.Name},
			{Key: "Version", Value: info.Version},
			{Key: "Commit", Value: info.Commit},
			{Key: "Built", Value: info.Date},
			{Key: "Go", Value: info.GoVersion},
			{Key: "Platform", Value: info.Platform},
		})
	}
	if err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

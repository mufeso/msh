package app

import (
	"fmt"

	"example.com/msh/internal/buildinfo"
)

const versionUsage = `Usage: msh version [--short]

Prints the msh version, source commit and build date.
`

func versionCommand() *command {
	return &command{
		name:    "version",
		summary: "Print version information",
		usage:   versionUsage,
		run:     runVersion,
	}
}

func runVersion(c *cli, args []string) int {
	fs := c.newFlagSet("version", versionUsage)
	short := fs.Bool("short", false, "print only the version number")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() > 0 {
		c.errorf("version takes no arguments")
		return exitUsage
	}

	if *short {
		fmt.Fprintln(c.stdout, buildinfo.Version)
		return exitOK
	}
	fmt.Fprintln(c.stdout, buildinfo.String())
	return exitOK
}

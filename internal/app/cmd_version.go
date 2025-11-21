package app

import (
	"fmt"

	"example.com/msh/internal/buildinfo"
)

func versionCommand() *command {
	return &command{
		name:    "version",
		summary: "Print version information",
		usage: `Usage: msh version

Prints the msh version, source commit and build date.
`,
		run: runVersion,
	}
}

func runVersion(c *cli, args []string) int {
	if len(args) > 0 {
		c.errorf("version takes no arguments")
		return exitUsage
	}
	fmt.Fprintln(c.stdout, buildinfo.String())
	return exitOK
}

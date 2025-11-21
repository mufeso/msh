package app

import "fmt"

func helpCommand() *command {
	return &command{
		name:    "help",
		summary: "Show help for msh or for one command",
		usage: `Usage: msh help [command]

Without arguments, lists all commands. With a command name, shows the
usage of that command.
`,
		run: runHelp,
	}
}

func runHelp(c *cli, args []string) int {
	switch len(args) {
	case 0:
		c.printUsage(c.stdout)
		return exitOK
	case 1:
		cmd, ok := lookup(args[0])
		if !ok {
			c.errorf("unknown command %q", args[0])
			return exitUsage
		}
		fmt.Fprint(c.stdout, cmd.usage)
		return exitOK
	default:
		c.errorf("help accepts at most one command name")
		return exitUsage
	}
}

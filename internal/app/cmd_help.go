package app

import (
	"fmt"

	"example.com/msh/internal/exitcode"
)

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

func runHelp(c *cli, args []string) exitcode.Code {
	switch len(args) {
	case 0:
		c.printUsage(c.stdout)
		return exitcode.OK
	case 1:
		if args[0] == "shortcuts" {
			fmt.Fprint(c.stdout, shortcutsHelp())
			return exitcode.OK
		}
		if args[0] == "exit-codes" {
			fmt.Fprintln(c.stdout, "Exit codes of msh (when servers differ: canceled > timeout > error > failed > ok):")
			for _, d := range exitcode.All() {
				fmt.Fprintf(c.stdout, "  %3d  %-10s %s\n", int(d.Code), d.Code, d.Meaning)
			}
			fmt.Fprintln(c.stdout, "msh shell passes the remote exit status through, like ssh.")
			return exitcode.OK
		}
		cmd, ok := lookup(args[0])
		if !ok {
			c.errorf("unknown command %q", args[0])
			return exitcode.Usage
		}
		fmt.Fprint(c.stdout, cmd.usage)
		return exitcode.OK
	default:
		c.errorf("help accepts at most one command name")
		return exitcode.Usage
	}
}

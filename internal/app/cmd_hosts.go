package app

import "fmt"

const hostsUsage = `Usage: msh hosts <subcommand> [flags]

Subcommands:
  list    List servers from the inventory
  check   Check SSH connectivity to servers
`

func hostsCommand() *command {
	return &command{
		name:    "hosts",
		summary: "List and check inventory servers",
		usage:   hostsUsage,
		run:     runHosts,
	}
}

// runHosts dispatches to a subcommand. args[0] is the subcommand name and
// args[1:] are its own arguments.
func runHosts(c *cli, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, hostsUsage)
		return exitUsage
	}

	switch args[0] {
	case "list":
		return runHostsList(c, args[1:])
	case "check":
		return runHostsCheck(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, hostsUsage)
		return exitOK
	default:
		c.errorf("unknown hosts subcommand %q", args[0])
		fmt.Fprint(c.stderr, hostsUsage)
		return exitUsage
	}
}

func runHostsList(c *cli, args []string) int {
	fmt.Fprintln(c.stdout, "No inventory configured yet.")
	return exitOK
}

func runHostsCheck(c *cli, args []string) int {
	fmt.Fprintln(c.stdout, "No inventory configured yet; nothing to check.")
	return exitOK
}

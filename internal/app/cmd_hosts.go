package app

import "fmt"

const hostsUsage = `Usage: msh hosts <subcommand> [flags]

Subcommands:
  list    List servers from the inventory
  check   Check SSH connectivity to servers
`

const hostsListUsage = `Usage: msh hosts list [flags]

Lists servers from the inventory.
`

const hostsCheckUsage = `Usage: msh hosts check [flags]

Checks SSH connectivity to servers.
`

func hostsCommand() *command {
	return &command{
		name:    "hosts",
		summary: "List and check inventory servers",
		usage:   hostsUsage,
		run:     runHosts,
	}
}

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
	fs := c.newFlagSet("hosts list", hostsListUsage)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	fmt.Fprintln(c.stdout, "No inventory configured yet.")
	return exitOK
}

func runHostsCheck(c *cli, args []string) int {
	fs := c.newFlagSet("hosts check", hostsCheckUsage)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	fmt.Fprintln(c.stdout, "No inventory configured yet; nothing to check.")
	return exitOK
}

package app

import (
	"fmt"

	"example.com/msh/internal/exitcode"
)

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

func runHosts(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, hostsUsage)
		return exitcode.Usage
	}

	switch args[0] {
	case "list":
		return runHostsList(c, args[1:])
	case "check":
		return runHostsCheck(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, hostsUsage)
		return exitcode.OK
	default:
		c.errorf("unknown hosts subcommand %q", args[0])
		fmt.Fprint(c.stderr, hostsUsage)
		return exitcode.Usage
	}
}

func runHostsList(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("hosts list", hostsListUsage)
	opts.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if err := opts.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	c.verbosef(&opts, "inventory file: %q", opts.inventoryPath)
	fmt.Fprintln(c.stdout, "No inventory configured yet.")
	return exitcode.OK
}

func runHostsCheck(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("hosts check", hostsCheckUsage)
	opts.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if err := opts.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	c.verbosef(&opts, "inventory file: %q", opts.inventoryPath)
	fmt.Fprintln(c.stdout, "No inventory configured yet; nothing to check.")
	return exitcode.OK
}

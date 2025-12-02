package app

import (
	"fmt"
	"strconv"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
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
	var sel selectionOptions
	fs := c.newFlagSet("hosts list", hostsListUsage)
	opts.register(fs)
	sel.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if err := opts.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if err := sel.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	c.verbosef(&opts, "selection: all=%v hosts=%v groups=%v tags=%v", sel.all, sel.hosts.String(), sel.groups.String(), sel.tags.String())
	inv, code := c.loadInventory(&opts)
	if inv == nil {
		return code
	}
	if err := c.writeServers(opts.format, inv.Servers); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

// writeServers prints servers as a table or as JSON.
func (c *cli) writeServers(format output.Format, servers []inventory.Server) error {
	if format == output.JSON {
		// Encode an empty slice as [] rather than null.
		if servers == nil {
			servers = []inventory.Server{}
		}
		return output.WriteJSON(c.stdout, servers)
	}
	rows := make([][]string, 0, len(servers))
	for _, s := range servers {
		rows = append(rows, []string{
			s.Name, s.Address, strconv.Itoa(s.Port), s.User, strings.Join(s.Groups, ","),
		})
	}
	return output.Table(c.stdout, []string{"NAME", "ADDRESS", "PORT", "USER", "GROUPS"}, rows)
}

func runHostsCheck(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	var sel selectionOptions
	fs := c.newFlagSet("hosts check", hostsCheckUsage)
	opts.register(fs)
	sel.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if err := opts.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	if err := sel.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	c.verbosef(&opts, "selection: all=%v hosts=%v groups=%v tags=%v", sel.all, sel.hosts.String(), sel.groups.String(), sel.tags.String())
	c.verbosef(&opts, "inventory file: %q", opts.inventoryPath)
	fmt.Fprintln(c.stdout, "No inventory configured yet; nothing to check.")
	return exitcode.OK
}

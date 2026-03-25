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

const hostsCheckUsage = `Usage: msh hosts check [--host NAME | --group NAME | --tag K=V | --all] [flags]

Checks SSH connectivity and authentication (like "msh ping"). Without a
selection, all servers are checked.
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
	if err := sel.validate(); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	c.verbosef(&opts, "selection: all=%v hosts=%v groups=%v tags=%v", sel.all, sel.hosts.String(), sel.groups.String(), sel.tags.String())
	st, code := c.load(&opts, true)
	if st == nil {
		return code
	}
	servers := st.inv.Servers
	if !sel.isEmpty() {
		var err error
		servers, err = st.inv.Select(sel.selector())
		if err != nil {
			c.errorf("%v", err)
			return exitcode.Usage
		}
	}
	if err := c.writeServers(opts.format, servers); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}

// writeServers prints servers as a table or as JSON.
func (c *cli) writeServers(format output.Format, servers []inventory.Server) error {
	rows := make([][]string, 0, len(servers))
	for _, s := range servers {
		rows = append(rows, []string{
			s.Name, s.Address, strconv.Itoa(s.Port), s.User, strings.Join(s.Groups, ","), via(s.Bastion),
		})
	}
	return output.WriteRows(c.stdout, format, []string{"NAME", "ADDRESS", "PORT", "USER", "GROUPS", "VIA"}, rows, servers)
}

// via shows the bastion name or "direct".
func via(bastion string) string {
	if bastion == "" {
		return "direct"
	}
	return bastion
}

func runHostsCheck(c *cli, args []string) exitcode.Code {
	return c.runShortcut(lookupShortcut("ping"), hostsCheckUsage, args, true)
}

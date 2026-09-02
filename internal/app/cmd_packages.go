package app

import (
	"flag"
	"fmt"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/pkgs"
)

const packagesUsage = `Usage: msh packages <subcommand> [flags]

Subcommands:
  list    List installed packages
  check   Check that packages are installed
  find    Find packages across servers
  compare Compare servers with a baseline server
  updates List pending updates

Works on servers with rpm or dpkg. msh only reads package data here.
`

const packagesListUsage = `Usage: msh packages list [PATTERN] ([user@]host | --host LIST | --group NAME | --tag K=V | --all) [flags]

Lists installed packages. PATTERN works like grep: a name only has to contain
it, and it is a regular expression. It can also be given with --filter. The
first plain word is the pattern, so write user@host, or use --host, for the
server.

Examples:
  msh packages list tmux root@db01
  msh packages list '^kernel' --group web
  msh packages list ssl --ignore-case --all
`

func packagesCommand() *command {
	return &command{
		name:    "packages",
		summary: "Check installed rpm and dpkg packages",
		usage:   packagesUsage,
		run:     runPackages,
	}
}

func runPackages(c *cli, args []string) exitcode.Code {
	if len(args) == 0 {
		fmt.Fprint(c.stderr, packagesUsage)
		return exitcode.Usage
	}
	switch args[0] {
	case "list":
		return runPackagesList(c, args[1:])
	case "check":
		return runPackagesCheck(c, args[1:])
	case "find":
		return runPackagesFind(c, args[1:])
	case "compare":
		return runPackagesCompare(c, args[1:])
	case "updates":
		return runPackagesUpdates(c, args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(c.stdout, packagesUsage)
		return exitcode.OK
	default:
		c.errorf("unknown packages subcommand %q", args[0])
		fmt.Fprint(c.stderr, packagesUsage)
		return exitcode.Usage
	}
}

// packagesFlags are the flags shared by the subcommands.
type packagesFlags struct {
	opts commonOptions
	sel  selectionOptions
	ro   remoteOptions
	grep grepFlags
}

func (f *packagesFlags) register(fs *flag.FlagSet) {
	f.opts.register(fs)
	f.sel.register(fs)
	f.sel.registerWhere(fs)
	f.ro.register(fs)
}

// collected is the package data of a fleet.
type collected struct {
	servers []inventory.Server
	found   map[string]*pkgs.List
	failed  []executor.Result
}

// collectPackages runs the collection. On failure it prints the error and
// returns nil and the exit code.
func (c *cli) collectPackages(name string, f *packagesFlags) (*collected, exitcode.Code) {
	p, code := c.prepareRun(name, &f.opts, &f.sel, &f.ro)
	if p == nil {
		return nil, code
	}
	defer p.stack.Close()
	ctx, cancel := c.runContext(p.st)
	defer cancel()
	found, failed := pkgs.Collect(ctx, p.stack.executor, p.servers, p.st.settings.Workers, p.st.settings.CommandTimeout)
	for name, l := range found {
		if l.Manager == pkgs.None {
			fmt.Fprintf(c.stderr, "msh: warning: %s: neither rpm nor dpkg found\n", name)
		}
	}
	return &collected{servers: p.servers, found: found, failed: failed}, exitcode.OK
}

// packageRow is the machine-readable form of one package on one server.
type packageRow struct {
	Server  string `json:"server"`
	Manager string `json:"manager"`
	pkgs.Package
}

func runPackagesList(c *cli, args []string) exitcode.Code {
	var f packagesFlags
	fs := c.newFlagSet("packages list", packagesListUsage)
	f.register(fs)
	f.grep.register(fs)
	filter := fs.String("filter", "", "show only names containing this `pattern`")
	words, code, ok := parseRemote(fs, &f.sel, args, 1)
	if !ok {
		return code
	}
	if len(words) == 1 {
		*filter = words[0]
	}
	match, ok := f.grep.compile(c, *filter)
	if !ok {
		return exitcode.Usage
	}
	data, code := c.collectPackages("packages list", &f)
	if data == nil {
		return code
	}
	items := []packageRow{}
	rows := [][]string{}
	for _, s := range data.servers {
		l, ok := data.found[s.Name]
		if !ok {
			continue
		}
		for _, p := range l.Packages {
			if match.Match(p.Name) {
				items = append(items, packageRow{Server: s.Name, Manager: l.Manager, Package: p})
				rows = append(rows, []string{s.Name, p.Name, p.Version, p.Arch})
			}
		}
	}
	if err := output.WriteRows(c.stdout, f.opts.format, []string{"SERVER", "PACKAGE", "VERSION", "ARCH"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return c.reportFailures(data.failed)
}

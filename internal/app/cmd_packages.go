package app

import (
	"flag"
	"fmt"
	"path"

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

Works on servers with rpm or dpkg. msh only reads package data here.
`

const packagesListUsage = `Usage: msh packages list (--host NAME | --group NAME | --tag K=V | --all) [--filter PATTERN] [flags]

Lists installed packages. PATTERN is a shell-style pattern: 'kernel*'.
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
	filter := fs.String("filter", "*", "show only names matching this shell-style `pattern`")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if _, err := path.Match(*filter, ""); err != nil {
		c.errorf("--filter: %v", err)
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
			if ok, _ := path.Match(*filter, p.Name); ok {
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

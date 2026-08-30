package app

import (
	"fmt"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/pkgs"
)

const packagesCompareUsage = `Usage: msh packages compare (--group NAME | --all | ...) [--baseline SERVER] [flags]

Compares the installed packages of each server with a baseline server
(default: the first selected). Exits 1 when servers differ.
`

// compareRow is one difference on one server.
type compareRow struct {
	Server string `json:"server"`
	pkgs.Difference
}

func runPackagesCompare(c *cli, args []string) exitcode.Code {
	var f packagesFlags
	fs := c.newFlagSet("packages compare", packagesCompareUsage)
	f.register(fs)
	baseline := fs.String("baseline", "", "compare with this `server` (default: the first selected)")
	if _, code, ok := parseRemote(fs, &f.sel, args, 0); !ok {
		return code
	}
	data, code := c.collectPackages("packages compare", &f)
	if data == nil {
		return code
	}
	name := *baseline
	if name == "" {
		name = data.servers[0].Name
	}
	base, ok := data.found[name]
	if !ok {
		c.reportFailures(data.failed)
		c.errorf("packages compare: no package data for baseline %q (is it selected and reachable?)", name)
		return exitcode.Usage
	}

	items := []compareRow{}
	rows := [][]string{}
	for _, s := range data.servers {
		l, ok := data.found[s.Name]
		if !ok || s.Name == name {
			continue
		}
		if l.Manager != base.Manager {
			fmt.Fprintf(c.stderr, "msh: warning: %s uses %s, %s uses %s; versions will not match\n", s.Name, l.Manager, name, base.Manager)
		}
		for _, d := range pkgs.Diff(base, l) {
			items = append(items, compareRow{Server: s.Name, Difference: d})
			rows = append(rows, []string{s.Name, d.Package, orDash(d.Baseline), orDash(d.Version)})
		}
	}
	if err := output.WriteRows(c.stdout, f.opts.format, []string{"SERVER", "PACKAGE", "BASELINE", "VERSION"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	if code := c.reportFailures(data.failed); code != exitcode.OK {
		return code
	}
	if len(items) > 0 {
		return exitcode.Failure
	}
	return exitcode.OK
}

// orDash shows "not installed" in tables.
func orDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

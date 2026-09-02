package app

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const packagesFindUsage = `Usage: msh packages find PATTERN ([user@]host | --host LIST | --group NAME | --tag K=V | --all) [flags]

Finds packages by name across servers and groups them by version.
PATTERN works like grep: a name only has to contain it, and it is a regular
expression: ssl, '^open', 'ssl$'.
`

// foundPackage is one package version and the servers that have it.
type foundPackage struct {
	Package string   `json:"package"`
	Version string   `json:"version"`
	Count   int      `json:"count"`
	Servers []string `json:"servers"`
}

func runPackagesFind(c *cli, args []string) exitcode.Code {
	var f packagesFlags
	fs := c.newFlagSet("packages find", packagesFindUsage)
	f.register(fs)
	f.grep.register(fs)
	patterns, code, ok := parseRemote(fs, &f.sel, args, 1)
	if !ok {
		return code
	}
	if len(patterns) != 1 {
		c.errorf("packages find: exactly one pattern is required")
		return exitcode.Usage
	}
	match, ok := f.grep.compile(c, patterns[0])
	if !ok {
		return exitcode.Usage
	}
	data, code := c.collectPackages("packages find", &f)
	if data == nil {
		return code
	}

	// A struct of comparable fields can be a map key.
	type key struct{ name, version string }
	servers := map[key][]string{}
	for _, s := range data.servers {
		l, ok := data.found[s.Name]
		if !ok {
			continue
		}
		for _, p := range l.Packages {
			if match.Match(p.Name) {
				k := key{p.Name, p.Version}
				if !slices.Contains(servers[k], s.Name) {
					servers[k] = append(servers[k], s.Name)
				}
			}
		}
	}
	items := make([]foundPackage, 0, len(servers))
	for k, names := range servers {
		items = append(items, foundPackage{Package: k.name, Version: k.version, Count: len(names), Servers: names})
	}
	slices.SortFunc(items, func(a, b foundPackage) int {
		if n := cmp.Compare(a.Package, b.Package); n != 0 {
			return n
		}
		return cmp.Compare(a.Version, b.Version)
	})
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Package, it.Version, strconv.Itoa(it.Count), strings.Join(it.Servers, ",")})
	}
	if err := output.WriteRows(c.stdout, f.opts.format, []string{"PACKAGE", "VERSION", "COUNT", "SERVERS"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return c.reportFailures(data.failed)
}

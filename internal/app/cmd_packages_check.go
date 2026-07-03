package app

import (
	"example.com/msh/internal/cliflag"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/pkgs"
)

const packagesCheckUsage = `Usage: msh packages check (--host NAME | --group NAME | --tag K=V | --all) --package NAME[,NAME...] [flags]

Checks each package on each server. Exits 1 if a requirement is not met.
`

// checkRow is one package requirement on one server.
type checkRow struct {
	Server      string `json:"server"`
	Requirement string `json:"requirement"`
	pkgs.CheckResult
}

func runPackagesCheck(c *cli, args []string) exitcode.Code {
	var f packagesFlags
	var names cliflag.StringList
	fs := c.newFlagSet("packages check", packagesCheckUsage)
	f.register(fs)
	fs.Var(&names, "package", "package `requirement` (repeatable, comma-separated)")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if len(names) == 0 {
		c.errorf("packages check: --package is required")
		return exitcode.Usage
	}
	specs := make([]pkgs.Spec, 0, len(names))
	for _, n := range names {
		s, err := pkgs.ParseSpec(n)
		if err != nil {
			c.errorf("--package: %v", err)
			return exitcode.Usage
		}
		specs = append(specs, s)
	}
	data, code := c.collectPackages("packages check", &f)
	if data == nil {
		return code
	}
	return c.reportChecks(&f, data, specs)
}

// reportChecks evaluates and prints the requirements.
func (c *cli) reportChecks(f *packagesFlags, data *collected, specs []pkgs.Spec) exitcode.Code {
	items := []checkRow{}
	rows := [][]string{}
	problems := 0
	for _, s := range data.servers {
		l, ok := data.found[s.Name]
		if !ok {
			continue
		}
		for _, spec := range specs {
			r := spec.Check(l)
			if r.Status != pkgs.StatusOK {
				problems++
			}
			items = append(items, checkRow{Server: s.Name, Requirement: spec.String(), CheckResult: r})
			version := r.Version
			if version == "" && f.opts.format == output.Text {
				version = "-"
			}
			rows = append(rows, []string{s.Name, spec.String(), r.Status, version})
		}
	}
	if err := output.WriteRows(c.stdout, f.opts.format, []string{"SERVER", "REQUIREMENT", "STATUS", "INSTALLED"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	if code := c.reportFailures(data.failed); code != exitcode.OK {
		return code
	}
	if problems > 0 {
		return exitcode.Failure
	}
	return exitcode.OK
}

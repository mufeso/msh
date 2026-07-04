package app

import (
	"os"

	"example.com/msh/internal/cliflag"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/pkgs"
)

const packagesCheckUsage = `Usage: msh packages check (--host NAME | --group NAME | --tag K=V | --all) (--package REQ[,REQ...] | --file FILE) [flags]

Checks requirements on each server and exits 1 if one is not met.
A requirement is NAME, NAME>=VERSION (also =, !=, <, <=, >) or !NAME, which
means the package must not be installed. Quote requirements in the shell.
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
	file := fs.String("file", "", "read requirements from this `file`, one per line")
	onlyProblems := fs.Bool("only-problems", false, "show only requirements that are not met")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if len(names) == 0 && *file == "" {
		c.errorf("packages check: use --package or --file")
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
	if *file != "" {
		from, code := c.readBaseline(*file)
		if from == nil {
			return code
		}
		specs = append(specs, from...)
	}
	data, code := c.collectPackages("packages check", &f)
	if data == nil {
		return code
	}
	return c.reportChecks(&f, data, specs, *onlyProblems)
}

// readBaseline reads a requirements file. On failure it prints the error
// and returns nil and the exit code.
func (c *cli) readBaseline(path string) ([]pkgs.Spec, exitcode.Code) {
	f, err := os.Open(path)
	if err != nil {
		c.errorf("%v", err)
		return nil, exitcode.IO
	}
	defer f.Close()
	specs, err := pkgs.ParseBaseline(f)
	if err != nil {
		c.errorf("%s: %v", path, err)
		return nil, exitcode.Config
	}
	if len(specs) == 0 {
		c.errorf("%s: no requirements", path)
		return nil, exitcode.Config
	}
	return specs, exitcode.OK
}

// reportChecks evaluates and prints the requirements.
func (c *cli) reportChecks(f *packagesFlags, data *collected, specs []pkgs.Spec, onlyProblems bool) exitcode.Code {
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
			} else if onlyProblems {
				continue
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

package app

import (
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/pkgs"
)

const packagesUpdatesUsage = `Usage: msh packages updates (--host NAME | --group NAME | --tag K=V | --all) [flags]

Lists pending updates (dnf, yum or apt). Nothing is installed. apt shows
what its last "apt update" found.
`

// updateRow is one pending update on one server.
type updateRow struct {
	Server string `json:"server"`
	pkgs.Update
}

func runPackagesUpdates(c *cli, args []string) exitcode.Code {
	var f packagesFlags
	fs := c.newFlagSet("packages updates", packagesUpdatesUsage)
	f.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	p, code := c.prepareRun("packages updates", &f.opts, &f.sel, &f.ro)
	if p == nil {
		return code
	}
	defer p.stack.Close()
	ctx, cancel := c.runContext(p.st)
	defer cancel()
	found, failed := pkgs.CollectUpdates(ctx, p.stack.executor, p.servers, p.st.settings.Workers, p.st.settings.CommandTimeout)

	items := []updateRow{}
	rows := [][]string{}
	for _, s := range p.servers {
		for _, u := range found[s.Name] {
			items = append(items, updateRow{Server: s.Name, Update: u})
			rows = append(rows, []string{s.Name, u.Name, u.Version, u.Source})
		}
	}
	if err := output.WriteRows(c.stdout, f.opts.format, []string{"SERVER", "PACKAGE", "NEW VERSION", "SOURCE"}, rows, items); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return c.reportFailures(failed)
}

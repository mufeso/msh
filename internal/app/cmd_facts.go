package app

import (
	"fmt"

	"example.com/msh/internal/config"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/facts"
	"example.com/msh/internal/output"
)

const factsUsage = `Usage: msh facts (--host NAME | --group NAME | --tag K=V | --all) [flags]

Collects read-only facts from servers: operating system, kernel,
architecture, CPUs, memory, root disk usage, package manager.
`

func factsCommand() *command {
	return &command{
		name:    "facts",
		summary: "Collect and show server facts",
		usage:   factsUsage,
		run:     runFacts,
	}
}

// factsView is the JSON shape of one server's facts.
type factsView struct {
	Server string      `json:"server"`
	Facts  facts.Facts `json:"facts"`
}

func runFacts(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	var sel selectionOptions
	var ro remoteOptions
	fs := c.newFlagSet("facts", factsUsage)
	opts.register(fs)
	sel.register(fs)
	sel.registerWhere(fs)
	ro.register(fs)
	refresh := fs.Bool("refresh", false, "ignore cached facts")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	p, code := c.prepareRun("facts", &opts, &sel, &ro)
	if p == nil {
		return code
	}
	defer p.stack.Close()

	ctx, cancel := c.runContext(p.st)
	defer cancel()
	g := c.gatherer(p.st, p.stack.executor)
	g.Refresh = *refresh
	found, failed := g.Gather(ctx, p.servers)

	views := make([]factsView, 0, len(found))
	rows := make([][]string, 0, len(found))
	for _, s := range p.servers {
		f, ok := found[s.Name]
		if !ok {
			continue
		}
		views = append(views, factsView{Server: s.Name, Facts: f})
		rows = append(rows, []string{s.Name, f["os_family"], f["os_id"], f["os_version"], f["kernel"], f["arch"], f["cpus"], f["memory_mb"], f["disk_root_pct"], f["package_manager"]})
	}
	headers := []string{"NAME", "FAMILY", "OS", "VERSION", "KERNEL", "ARCH", "CPUS", "MEMORY_MB", "DISK_PCT", "PACKAGES"}
	if err := output.WriteRows(c.stdout, opts.format, headers, rows, views); err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return c.reportFailures(failed)
}

// gatherer builds a fact gatherer from the settings.
func (c *cli) gatherer(st *setup, exec executor.Executor) *facts.Gatherer {
	return &facts.Gatherer{
		Exec: exec, Workers: st.settings.Workers, Timeout: st.settings.CommandTimeout,
		CacheDir: config.CacheDir(c.lookupEnv), TTL: st.settings.FactsTTL,
	}
}

// reportFailures prints servers that gave no data and returns the exit
// code they imply.
func (c *cli) reportFailures(failed []executor.Result) exitcode.Code {
	for _, r := range failed {
		fmt.Fprintf(c.stderr, "msh: %s: %s", r.Server, r.Status)
		if r.Err != nil {
			fmt.Fprintf(c.stderr, ": %v", r.Err)
		}
		fmt.Fprintln(c.stderr)
	}
	return exitCodeForResults(failed)
}

package facts

import (
	"context"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// Gatherer collects facts from servers.
type Gatherer struct {
	Exec    executor.Executor
	Workers int
	Timeout time.Duration
}

// Gather returns the facts of every server that answered and the results
// of the servers that did not.
func (g *Gatherer) Gather(ctx context.Context, servers []inventory.Server) (map[string]Facts, []executor.Result) {
	found := make(map[string]Facts, len(servers))
	var failed []executor.Result
	req := executor.Request{Command: Command, Stdin: []byte(Script), Timeout: g.Timeout}
	report := runner.Run(ctx, g.Exec, servers, req, runner.Options{Workers: g.Workers}, nil)
	for _, r := range report.Results {
		if r.Status == executor.StatusOK {
			found[r.Server] = Parse(r.Stdout)
		} else {
			failed = append(failed, r)
		}
	}
	return found, failed
}

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

	CacheDir string           // "" disables the cache
	TTL      time.Duration    // how long cached facts stay valid
	Refresh  bool             // ignore cached facts
	Now      func() time.Time // the clock; nil means time.Now
}

// Gather returns the facts of every server that answered and the results
// of the servers that did not.
func (g *Gatherer) Gather(ctx context.Context, servers []inventory.Server) (map[string]Facts, []executor.Result) {
	found := make(map[string]Facts, len(servers))
	var missing []inventory.Server
	for _, s := range servers {
		if f, ok := g.cached(s); ok {
			found[s.Name] = f
		} else {
			missing = append(missing, s)
		}
	}
	if len(missing) == 0 {
		return found, nil
	}

	var failed []executor.Result
	byName := make(map[string]inventory.Server, len(missing))
	for _, s := range missing {
		byName[s.Name] = s
	}
	req := executor.Request{Command: Command, Stdin: []byte(Script), Timeout: g.Timeout}
	report := runner.Run(ctx, g.Exec, missing, req, runner.Options{Workers: g.Workers}, nil)
	for _, r := range report.Results {
		if r.Status != executor.StatusOK {
			failed = append(failed, r)
			continue
		}
		f := Parse(r.Stdout)
		found[r.Server] = f
		g.store(byName[r.Server], f)
	}
	return found, failed
}

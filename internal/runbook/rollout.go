package runbook

import (
	"context"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// Rollout runs servers batch by batch: a batch must finish before the
// next starts, so only part of the fleet is affected at any moment.
type Rollout struct {
	BatchSize int // 0 = everything in one batch
}

// Run executes ex on all servers and returns one merged report, with the
// results in selection order.
func (r Rollout) Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, opts runner.Options, emit func(executor.Result)) runner.Report {
	start := time.Now()
	results := make([]executor.Result, 0, len(servers))
	for _, batch := range Batches(servers, r.BatchSize) {
		report := runner.Run(ctx, ex, batch, executor.Request{}, opts, emit)
		results = append(results, report.Results...)
	}
	summary := runner.Summarize(results)
	summary.Duration = time.Since(start)
	return runner.Report{Results: results, Summary: summary}
}

// NewRollout resolves the runbook's settings for a selection of total
// servers.
func (rb *Runbook) NewRollout(total int) (Rollout, error) {
	var r Rollout
	if rb.Batch != "" {
		a, err := ParseAmount(rb.Batch)
		if err != nil {
			return r, err
		}
		r.BatchSize = a.Of(total)
	}
	return r, nil
}

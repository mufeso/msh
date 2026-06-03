package runbook

import (
	"context"
	"fmt"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// Rollout runs servers batch by batch: a batch must finish before the
// next starts, so only part of the fleet is affected at any moment.
type Rollout struct {
	BatchSize   int // 0 = everything in one batch
	MaxFailures int // failed servers tolerated before stopping; negative = no limit
}

// Run executes ex on all servers and returns one merged report, with the
// results in selection order.
func (r Rollout) Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, opts runner.Options, emit func(executor.Result)) runner.Report {
	start := time.Now()
	results := make([]executor.Result, 0, len(servers))
	failures := 0
	stopReason := ""
	for _, batch := range Batches(servers, r.BatchSize) {
		if stopReason != "" {
			results = append(results, skipBatch(batch, stopReason, emit)...)
			continue
		}
		report := runner.Run(ctx, ex, batch, executor.Request{}, opts, emit)
		results = append(results, report.Results...)
		failures += report.Summary.Total - report.Summary.OK
		if r.MaxFailures >= 0 && failures > r.MaxFailures {
			stopReason = fmt.Sprintf("rollout stopped: %d failed servers (limit %d)", failures, r.MaxFailures)
		}
	}
	summary := runner.Summarize(results)
	summary.Duration = time.Since(start)
	return runner.Report{Results: results, Summary: summary}
}

// skipBatch reports every server of a batch as skipped.
func skipBatch(batch []inventory.Server, reason string, emit func(executor.Result)) []executor.Result {
	out := make([]executor.Result, 0, len(batch))
	for _, s := range batch {
		res := runner.Skipped(s, reason)
		out = append(out, res)
		if emit != nil {
			emit(res)
		}
	}
	return out
}

// Overrides are rollout settings from the command line; empty values
// keep the runbook's settings.
type Overrides struct {
	Batch       string
	MaxFailures string
}

// WithOverrides returns a copy of the runbook with o applied.
func (rb Runbook) WithOverrides(o Overrides) (*Runbook, error) {
	if err := validateRollout(o.Batch, o.MaxFailures, 0); err != nil {
		return nil, err
	}
	if o.Batch != "" {
		rb.Batch = o.Batch
	}
	if o.MaxFailures != "" {
		rb.MaxFailures = o.MaxFailures
	}
	return &rb, nil
}

// NewRollout resolves the runbook's settings for a selection of total
// servers. Without batches there is no limit on failures, as for any
// parallel run; with batches the default limit is 0.
func (rb *Runbook) NewRollout(total int) (Rollout, error) {
	r := Rollout{MaxFailures: -1}
	if rb.Batch != "" {
		a, err := ParseAmount(rb.Batch)
		if err != nil {
			return r, err
		}
		r.BatchSize = a.Of(total)
		r.MaxFailures = 0
	}
	if rb.MaxFailures != "" {
		a, err := ParseAmount(rb.MaxFailures)
		if err != nil {
			return r, err
		}
		r.MaxFailures = a.Of(total)
	}
	return r, nil
}

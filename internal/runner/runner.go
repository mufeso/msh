// Package runner runs one request on many servers.
//
// It only knows the executor.Executor interface, so the same code drives
// SSH execution, local execution and test fakes.
package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Options control a run.
type Options struct {
	// FailFast stops starting new servers after the first server that did
	// not succeed. Servers already running are not interrupted.
	FailFast bool
}

// Run executes req on each server, one after another, and returns a
// report with the results in selection order. emit, if not nil, is called
// with each result as soon as it is available.
//
// Once ctx is done no further server is started; the remaining servers
// get a "not started" result, so every server is always reported.
func Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, opts Options, emit func(executor.Result)) Report {
	start := time.Now()
	// A child context lets fail-fast stop the run without touching the
	// caller's context. cancel must always be called: defer it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]executor.Result, 0, len(servers))
	for _, s := range servers {
		var r executor.Result
		if err := ctx.Err(); err != nil {
			r = notStarted(s, err)
		} else {
			r = ex.Run(ctx, s, req)
		}
		results = append(results, r)
		if emit != nil {
			emit(r)
		}
		if opts.FailFast && r.Status != executor.StatusOK {
			cancel()
		}
	}
	summary := Summarize(results)
	summary.Duration = time.Since(start)
	return Report{Results: results, Summary: summary}
}

// notStarted is the result for a server that was never contacted.
func notStarted(s inventory.Server, cause error) executor.Result {
	kind, status := errs.KindCanceled, executor.StatusCanceled
	if errors.Is(cause, context.DeadlineExceeded) {
		kind, status = errs.KindGlobalTimeout, executor.StatusTimeout
	}
	return executor.Result{
		Server:   s.Name,
		Address:  s.Address,
		Status:   status,
		ExitCode: -1,
		Err:      errs.New(kind, s.Name, fmt.Errorf("not started: %w", cause)),
		Started:  time.Now(),
	}
}

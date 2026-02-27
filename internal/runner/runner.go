// Package runner runs one request on many servers.
//
// It only knows the executor.Executor interface, so the same code drives
// SSH execution, local execution and test fakes.
package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

	// Parallel runs all servers at the same time.
	Parallel bool
}

// indexed carries a result together with its position in the selection.
type indexed struct {
	index  int
	result executor.Result
}

// Run executes req on the servers and returns a report with the results
// in selection order. emit, if not nil, is called with each result as
// soon as it is available, always from the caller's goroutine.
func Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, opts Options, emit func(executor.Result)) Report {
	start := time.Now()
	var results []executor.Result
	if opts.Parallel {
		results = runParallel(ctx, ex, servers, req, emit)
	} else {
		results = runSequential(ctx, ex, servers, req, opts, emit)
	}
	summary := Summarize(results)
	summary.Duration = time.Since(start)
	return Report{Results: results, Summary: summary}
}

// runParallel starts one goroutine per server. The goroutines only send;
// this function is the only receiver, so no locks are needed.
func runParallel(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, emit func(executor.Result)) []executor.Result {
	out := make(chan indexed)
	var wg sync.WaitGroup
	for i, s := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out <- indexed{index: i, result: runOne(ctx, ex, s, req)}
		}()
	}
	// Close the channel once every sender is done, so the range below
	// ends. This must run in its own goroutine: wg.Wait would otherwise
	// block forever, because the senders wait for us to receive.
	go func() {
		wg.Wait()
		close(out)
	}()

	results := make([]executor.Result, len(servers))
	for ir := range out {
		results[ir.index] = ir.result
		if emit != nil {
			emit(ir.result)
		}
	}
	return results
}

// runSequential runs the servers one after another.
func runSequential(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, opts Options, emit func(executor.Result)) []executor.Result {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]executor.Result, 0, len(servers))
	for _, s := range servers {
		var r executor.Result
		if err := ctx.Err(); err != nil {
			r = notStarted(s, err)
		} else {
			r = runOne(ctx, ex, s, req)
		}
		results = append(results, r)
		if emit != nil {
			emit(r)
		}
		if opts.FailFast && r.Status != executor.StatusOK {
			cancel()
		}
	}
	return results
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

// runOne runs req on one server under its own child context. Canceling
// the child right after the run releases anything tied to it.
func runOne(ctx context.Context, ex executor.Executor, s inventory.Server, req executor.Request) executor.Result {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return ex.Run(ctx, s, req)
}

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
	// Workers is the maximum number of servers handled at the same time.
	// 1 (or less) means sequential execution.
	Workers int

	// FailFast stops starting new servers after the first server that did
	// not succeed. Servers already running are not interrupted.
	FailFast bool
}

// indexed carries a result together with its position in the selection.
type indexed struct {
	index  int
	result executor.Result
}

// Run executes req on the servers with a bounded worker pool and returns
// a report with the results in selection order. emit, if not nil, is
// called with each result as soon as it is available, always from the
// caller's goroutine.
//
//	dispatcher --jobs--> worker 1..N --out--> collector (this function)
//
// Two contexts are involved: ctx cancels running work (CTRL+C, global
// timeout); sched only stops handing out new servers (fail-fast, or ctx
// itself ending). Servers that are never started get a "not started"
// result, so every server is always reported.
func Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, opts Options, emit func(executor.Result)) Report {
	start := time.Now()
	sched, stopScheduling := context.WithCancel(ctx)
	defer stopScheduling()

	jobs := make(chan int)
	out := make(chan indexed)
	var wg sync.WaitGroup

	for w := 0; w < workerCount(opts.Workers, len(servers)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				var r executor.Result
				if err := sched.Err(); err != nil {
					// Scheduling stopped between dispatch and pickup.
					r = notStarted(servers[i], cause(ctx, err))
				} else {
					r = runOne(ctx, ex, servers[i], req)
				}
				// The worker decides about fail-fast itself, before it can
				// pick up another job; that makes the behavior exact.
				if opts.FailFast && r.Status != executor.StatusOK {
					stopScheduling()
				}
				out <- indexed{index: i, result: r}
			}
		}()
	}

	wg.Add(1)
	go func() { // dispatcher
		defer wg.Done()
		defer close(jobs)
		for i := range servers {
			if err := sched.Err(); err != nil {
				out <- indexed{index: i, result: notStarted(servers[i], cause(ctx, err))}
				continue
			}
			jobs <- i
		}
	}()

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
	summary := Summarize(results)
	summary.Duration = time.Since(start)
	return Report{Results: results, Summary: summary}
}

// cause explains why scheduling stopped: the caller's context ending
// (cancellation or global timeout), or fail-fast.
func cause(ctx context.Context, schedErr error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("fail-fast after an earlier failure: %w", schedErr)
}

// workerCount clamps the configured number of workers to [1, servers].
func workerCount(configured, servers int) int {
	return max(1, min(configured, servers))
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

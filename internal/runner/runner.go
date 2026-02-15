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

// Run executes req on each server, one after another, and returns the
// results in selection order. emit, if not nil, is called with each
// result as soon as it is available.
//
// Once ctx is done no further server is started; the remaining servers
// get a "not started" result, so every server is always reported.
func Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, emit func(executor.Result)) []executor.Result {
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

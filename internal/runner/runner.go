// Package runner runs one request on many servers.
//
// It only knows the executor.Executor interface, so the same code drives
// SSH execution, local execution and test fakes.
package runner

import (
	"context"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Run executes req on each server, one after another, and returns the
// results in selection order. emit, if not nil, is called with each
// result as soon as it is available.
func Run(ctx context.Context, ex executor.Executor, servers []inventory.Server, req executor.Request, emit func(executor.Result)) []executor.Result {
	results := make([]executor.Result, 0, len(servers))
	for _, s := range servers {
		r := ex.Run(ctx, s, req)
		results = append(results, r)
		if emit != nil {
			emit(r)
		}
	}
	return results
}

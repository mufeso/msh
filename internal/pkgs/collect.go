package pkgs

import (
	"context"
	"errors"
	"time"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// outputLimit fits the package list of any normal server.
const outputLimit = 64 << 20

// Collect reads the installed packages of each server. It returns the
// lists of servers that answered and the results of those that did not.
func Collect(ctx context.Context, ex executor.Executor, servers []inventory.Server, workers int, timeout time.Duration) (map[string]*List, []executor.Result) {
	req := executor.Request{Command: Command, Stdin: []byte(Script), Timeout: timeout, OutputLimit: outputLimit}
	report := runner.Run(ctx, ex, servers, req, runner.Options{Workers: workers}, nil)
	found := make(map[string]*List, len(servers))
	var failed []executor.Result
	for _, r := range report.Results {
		switch {
		case r.Status != executor.StatusOK:
			failed = append(failed, r)
		case r.StdoutTruncated:
			// A partial list would report packages as missing.
			r.Status = executor.StatusError
			r.Err = errs.New(errs.KindRemoteCommand, r.Server, errors.New("the package list was too long"))
			failed = append(failed, r)
		default:
			found[r.Server] = Parse(r.Stdout)
		}
	}
	return found, failed
}

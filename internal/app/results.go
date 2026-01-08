package app

import (
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
)

// statusPriority orders statuses from most to least significant for the
// overall exit code: a cancellation explains everything else, a timeout
// matters more than a non-zero exit, and so on.
var statusPriority = []executor.Status{
	executor.StatusCanceled,
	executor.StatusTimeout,
	executor.StatusError,
	executor.StatusFailed,
}

// exitCodeForResults returns the process exit code for a set of results.
func exitCodeForResults(results []executor.Result) exitcode.Code {
	seen := map[executor.Status]bool{}
	for _, r := range results {
		seen[r.Status] = true
	}
	for _, st := range statusPriority {
		if seen[st] {
			return exitCodeForStatus(st)
		}
	}
	return exitcode.OK
}

func exitCodeForStatus(st executor.Status) exitcode.Code {
	switch st {
	case executor.StatusOK:
		return exitcode.OK
	case executor.StatusFailed:
		return exitcode.Failure
	case executor.StatusTimeout:
		return exitcode.Timeout
	case executor.StatusCanceled:
		return exitcode.Canceled
	default:
		return exitcode.Internal
	}
}

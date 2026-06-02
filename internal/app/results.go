package app

import (
	"example.com/msh/internal/errs"
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
	executor.StatusSkipped,
}

// exitCodeForResults returns the process exit code for a set of results.
func exitCodeForResults(results []executor.Result) exitcode.Code {
	first := map[executor.Status]executor.Result{}
	for _, r := range results {
		if _, ok := first[r.Status]; !ok {
			first[r.Status] = r
		}
	}
	for _, st := range statusPriority {
		if r, ok := first[st]; ok {
			return exitCodeForResult(r)
		}
	}
	return exitcode.OK
}

func exitCodeForStatus(st executor.Status) exitcode.Code {
	switch st {
	case executor.StatusOK:
		return exitcode.OK
	case executor.StatusFailed, executor.StatusSkipped:
		return exitcode.Failure
	case executor.StatusTimeout:
		return exitcode.Timeout
	case executor.StatusCanceled:
		return exitcode.Canceled
	default:
		return exitcode.Internal
	}
}

// exitCodeForResult refines the code for one result: errors are mapped by
// their kind (connection, transfer, ...).
func exitCodeForResult(r executor.Result) exitcode.Code {
	if r.Status == executor.StatusError {
		return errs.ExitCode(errs.KindOf(r.Err))
	}
	return exitCodeForStatus(r.Status)
}

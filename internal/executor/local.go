package executor

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"example.com/msh/internal/inventory"
)

// LocalExecutor runs commands on the local machine with /bin/sh -c,
// which mirrors how sshd runs a remote command through the user's shell.
type LocalExecutor struct{}

// Run executes req.Command locally. The server only labels the result.
func (LocalExecutor) Run(ctx context.Context, server inventory.Server, req Request) Result {
	res := Result{Server: server.Name, Address: server.Address, ExitCode: -1, Started: time.Now()}

	// The command line is passed as ONE argument to sh -c. exec.Command
	// never builds a shell string from its arguments, so nothing else is
	// interpreted by a shell.
	// Derive a context that also ends when the timeout expires. cancel
	// must always be called to release the timer, hence the defer.
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// CommandContext kills the process when ctx is done.
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", req.Command)
	stop := setProcessGroup(cmd)
	defer stop()
	// Separate buffers: os/exec copies stdout and stderr from separate
	// goroutines, and LimitedBuffer is not safe for concurrent writes.
	stdout := NewLimitedBuffer(req.OutputLimit)
	stderr := NewLimitedBuffer(req.OutputLimit)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	res.Duration = time.Since(res.Started)
	res.Stdout, res.StdoutTruncated = stdout.String(), stdout.Truncated()
	res.Stderr, res.StderrTruncated = stderr.String(), stderr.Truncated()

	classify(&res, err)
	// If the context ended, that explains the failure better than the
	// "signal: killed" error from the process.
	if ctxErr := ctx.Err(); ctxErr != nil {
		res.Status, res.Err = statusForContext(ctxErr), ctxErr
	}
	return res
}

// statusForContext maps a context error to a Status.
func statusForContext(err error) Status {
	if errors.Is(err, context.DeadlineExceeded) {
		return StatusTimeout
	}
	return StatusCanceled
}

// classify sets Status, ExitCode and Err from the error returned by
// cmd.Run.
func classify(res *Result, err error) {
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		res.Status, res.ExitCode = StatusOK, 0
	case errors.As(err, &exitErr):
		// The command ran and exited non-zero. That is a normal outcome,
		// not an execution error, so Err stays nil. ExitCode returns -1
		// when the process was killed by a signal.
		res.Status, res.ExitCode = StatusFailed, exitErr.ExitCode()
	default:
		// The command could not be started at all.
		res.Status, res.Err = StatusError, err
	}
}

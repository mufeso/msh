package executor

import (
	"context"
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
	cmd := exec.Command("/bin/sh", "-c", req.Command)
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

	if err != nil {
		res.Status = StatusError
		res.Err = err
		return res
	}
	res.Status = StatusOK
	res.ExitCode = 0
	return res
}

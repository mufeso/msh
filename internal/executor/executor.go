// Package executor defines how msh runs a command on one server.
//
// The Executor interface has one method. Local execution (for testing and
// local tools), SSH execution and test fakes all implement it, so the code
// that runs commands on many servers never needs to know which one it uses.
package executor

import (
	"context"
	"io"
	"time"

	"example.com/msh/internal/inventory"
)

// Executor runs a command on one server.
//
// Run must:
//   - return when ctx is canceled or its deadline expires,
//   - release every resource it acquired before returning,
//   - report every outcome in the Result (it never panics on remote
//     failures and has no separate error return).
type Executor interface {
	Run(ctx context.Context, server inventory.Server, req Request) Result
}

// Request describes a command to run.
type Request struct {
	// Command is a command line. Like ssh, it is interpreted by a shell
	// on the target (the remote user's login shell, or /bin/sh locally).
	Command string

	// OutputLimit is the maximum number of bytes kept per output stream;
	// 0 means DefaultOutputLimit.
	OutputLimit int

	// Timeout limits how long the command may run; 0 means no limit
	// beyond the context passed to Run.
	Timeout time.Duration

	// Stdin, if not nil, is sent to the command's standard input. It is a
	// byte slice rather than an io.Reader so that each server gets its
	// own reader: one script can be streamed to many servers at once.
	Stdin []byte

	// Stream, if not nil, returns writers that receive the command's
	// output live, in addition to the captured Result fields. It is a
	// function because every server needs its own writers.
	Stream func(server inventory.Server) (stdout, stderr io.Writer)
}

// Outputs returns the writers an executor should connect to a process:
// the capture buffers, plus the live stream writers if requested.
func Outputs(server inventory.Server, req Request, stdout, stderr *LimitedBuffer) (io.Writer, io.Writer) {
	if req.Stream == nil {
		return stdout, stderr
	}
	liveOut, liveErr := req.Stream(server)
	// MultiWriter writes every chunk to each writer in turn.
	return io.MultiWriter(stdout, liveOut), io.MultiWriter(stderr, liveErr)
}

// Status is the outcome of running a command on one server.
type Status string

// Possible statuses.
const (
	StatusOK       Status = "ok"       // command ran and exited 0
	StatusFailed   Status = "failed"   // command ran and exited non-zero
	StatusTimeout  Status = "timeout"  // command or connection timed out
	StatusCanceled Status = "canceled" // the user canceled the run
	StatusError    Status = "error"    // the command could not be run
	StatusSkipped  Status = "skipped"  // a runbook step that did not run
)

// StepResult is the outcome of one step of a runbook.
type StepResult struct {
	Name       string `json:"name"`
	Status     Status `json:"status"`
	ExitCode   int    `json:"exit_code"`
	Attempts   int    `json:"attempts"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

// Result is what happened when a command ran on one server.
type Result struct {
	Server   string // inventory name
	Address  string // address that was used
	Status   Status // outcome
	ExitCode int    // process exit code; -1 when there is none

	Stdout          string // captured standard output
	Stderr          string // captured standard error
	StdoutTruncated bool   // stdout exceeded the output limit
	StderrTruncated bool   // stderr exceeded the output limit

	Steps []StepResult // per-step details of a runbook; nil for single commands

	Err      error         // why Status is not StatusOK or StatusFailed
	Started  time.Time     // when the attempt started
	Duration time.Duration // how long it took
}

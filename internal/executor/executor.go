// Package executor defines how msh runs a command on one server.
//
// The Executor interface has one method. Local execution (for testing and
// local tools), SSH execution and test fakes all implement it, so the code
// that runs commands on many servers never needs to know which one it uses.
package executor

import (
	"context"
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
)

// Result is what happened when a command ran on one server.
type Result struct {
	Server   string        // inventory name
	Address  string        // address that was used
	Status   Status        // outcome
	ExitCode int           // process exit code; -1 when there is none
	Stdout   string        // captured standard output
	Stderr   string        // captured standard error
	Err      error         // why Status is not StatusOK or StatusFailed
	Started  time.Time     // when the attempt started
	Duration time.Duration // how long it took
}

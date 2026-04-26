package runbook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Engine runs a plan on one server.
//
// It implements executor.Executor by wrapping another Executor, so
// everything built for single commands (the worker pool, cancellation,
// output sinks, exit codes) also works for whole runbooks. The request
// passed to Run is ignored: the plan says what to run.
type Engine struct {
	Exec           executor.Executor
	Steps          []PlannedStep
	DefaultTimeout time.Duration // used by steps without their own timeout
}

var _ executor.Executor = (*Engine)(nil)

// Run executes the steps in order and stops at the first failure.
func (e *Engine) Run(ctx context.Context, server inventory.Server, _ executor.Request) executor.Result {
	res := executor.Result{
		Server: server.Name, Address: server.Address,
		Status: executor.StatusOK, Started: time.Now(),
	}
	var stdout, stderr strings.Builder
	stopped := false
	for _, ps := range e.Steps {
		if stopped {
			res.Steps = append(res.Steps, executor.StepResult{Name: ps.Step.Name, Status: executor.StatusSkipped, ExitCode: -1})
			continue
		}
		r := e.runStep(ctx, server, ps)
		res.Steps = append(res.Steps, stepResult(ps, r))
		fmt.Fprintf(&stdout, "--- step %s: %s (exit %d)\n", ps.Step.Name, r.Status, r.ExitCode)
		stdout.WriteString(r.Stdout)
		stderr.WriteString(r.Stderr)
		if r.Status != executor.StatusOK {
			stopped = true
			res.Status, res.ExitCode = r.Status, r.ExitCode
			if r.Err != nil {
				res.Err = fmt.Errorf("step %q: %w", ps.Step.Name, r.Err)
			}
		}
	}
	res.Stdout, res.Stderr = stdout.String(), stderr.String()
	res.Duration = time.Since(res.Started)
	return res
}

// runStep runs one step.
func (e *Engine) runStep(ctx context.Context, server inventory.Server, ps PlannedStep) executor.Result {
	timeout := ps.Timeout
	if timeout == 0 {
		timeout = e.DefaultTimeout
	}
	return e.Exec.Run(ctx, server, executor.Request{Command: ps.Command, Stdin: ps.Stdin, Timeout: timeout})
}

// stepResult records the outcome of a step.
func stepResult(ps PlannedStep, r executor.Result) executor.StepResult {
	sr := executor.StepResult{
		Name: ps.Step.Name, Status: r.Status, ExitCode: r.ExitCode, Attempts: 1,
		Stdout: r.Stdout, Stderr: r.Stderr, DurationMS: r.Duration.Milliseconds(),
	}
	if r.Err != nil {
		sr.Error = r.Err.Error()
	}
	return sr
}

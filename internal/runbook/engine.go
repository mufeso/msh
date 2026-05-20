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
	ServerTimeout  time.Duration // total limit per server; 0 = none
	Progress       Progress      // optional: skips finished steps, records new ones
}

var _ executor.Executor = (*Engine)(nil)

// Run executes the steps in order and stops at the first failure.
func (e *Engine) Run(ctx context.Context, server inventory.Server, _ executor.Request) executor.Result {
	res := executor.Result{
		Server: server.Name, Address: server.Address,
		Status: executor.StatusOK, Started: time.Now(),
	}
	if e.ServerTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.ServerTimeout)
		defer cancel()
	}
	var stdout, stderr strings.Builder
	stopped := false
	done := 0
	if e.Progress != nil {
		done = e.Progress.Completed(server.Name)
	}
	for i, ps := range e.Steps {
		if i < done {
			fmt.Fprintf(&stdout, "--- step %s: already done\n", ps.Step.Name)
			res.Steps = append(res.Steps, executor.StepResult{Name: ps.Step.Name, Status: executor.StatusSkipped, ExitCode: -1})
			continue
		}
		if stopped {
			res.Steps = append(res.Steps, executor.StepResult{Name: ps.Step.Name, Status: executor.StatusSkipped, ExitCode: -1})
			continue
		}
		r, attempts := e.runStep(ctx, server, ps)
		sr := stepResult(ps, r)
		sr.Attempts = attempts
		res.Steps = append(res.Steps, sr)
		fmt.Fprintf(&stdout, "--- step %s: %s (exit %d)\n", ps.Step.Name, r.Status, r.ExitCode)
		stdout.WriteString(r.Stdout)
		stderr.WriteString(r.Stderr)
		tolerated := ps.Step.ContinueOnError && r.Status != executor.StatusCanceled
		if r.Status != executor.StatusOK && !tolerated {
			stopped = true
			res.Status, res.ExitCode = r.Status, r.ExitCode
			if r.Err != nil {
				res.Err = fmt.Errorf("step %q: %w", ps.Step.Name, r.Err)
			}
			continue
		}
		if e.Progress != nil {
			e.Progress.Mark(server.Name, i+1)
		}
	}
	res.Stdout, res.Stderr = stdout.String(), stderr.String()
	res.Duration = time.Since(res.Started)
	return res
}

// runStep runs one step, retrying as configured. It returns the last
// result and the number of attempts.
func (e *Engine) runStep(ctx context.Context, server inventory.Server, ps PlannedStep) (executor.Result, int) {
	timeout := ps.Timeout
	if timeout == 0 {
		timeout = e.DefaultTimeout
	}
	req := executor.Request{Command: ps.Command, Stdin: ps.Stdin, Timeout: timeout}
	var delay time.Duration
	if ps.Step.RetryDelay != nil {
		delay = time.Duration(*ps.Step.RetryDelay)
	}
	for attempt := 1; ; attempt++ {
		r := e.Exec.Run(ctx, server, req)
		if r.Status == executor.StatusOK || r.Status == executor.StatusCanceled ||
			attempt > ps.Step.Retries || !wait(ctx, delay) {
			return r, attempt
		}
	}
}

// wait sleeps for d and reports false if ctx ended first. NewTimer with
// Stop releases the timer at once, unlike time.After.
func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
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

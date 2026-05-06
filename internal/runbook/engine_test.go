package runbook

import (
	"context"
	"sync"
	"testing"
	"time"

	"example.com/msh/internal/config"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// scriptedExec answers by command and records what ran.
type scriptedExec struct {
	mu      sync.Mutex
	answers map[string][]executor.Status // per command: results in order; last repeats
	calls   []string
}

func (s *scriptedExec) Run(_ context.Context, server inventory.Server, req executor.Request) executor.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, req.Command)
	status := executor.StatusOK
	if list := s.answers[req.Command]; len(list) > 0 {
		status = list[0]
		if len(list) > 1 {
			s.answers[req.Command] = list[1:]
		}
	}
	code := 0
	if status != executor.StatusOK {
		code = 1
	}
	return executor.Result{Server: server.Name, Status: status, ExitCode: code, Stdout: req.Command + "\n"}
}

func plan(cmds ...string) []PlannedStep {
	out := make([]PlannedStep, len(cmds))
	for i, c := range cmds {
		out[i] = PlannedStep{Step: Step{Name: c}, Command: c}
	}
	return out
}

var srv = inventory.Server{Name: "s1", Address: "192.0.2.1"}

func TestEngineRunsStepsInOrder(t *testing.T) {
	ex := &scriptedExec{}
	r := (&Engine{Exec: ex, Steps: plan("a", "b", "c")}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusOK || len(r.Steps) != 3 {
		t.Fatalf("result = %+v", r)
	}
	if len(ex.calls) != 3 || ex.calls[0] != "a" || ex.calls[2] != "c" {
		t.Errorf("calls = %v", ex.calls)
	}
}

func TestEngineStopsAtFirstFailure(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"b": {executor.StatusFailed}}}
	r := (&Engine{Exec: ex, Steps: plan("a", "b", "c")}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusFailed || r.ExitCode != 1 {
		t.Errorf("result = %+v", r)
	}
	if len(ex.calls) != 2 {
		t.Errorf("calls = %v", ex.calls)
	}
	if r.Steps[2].Status != executor.StatusSkipped {
		t.Errorf("step c = %+v", r.Steps[2])
	}
}

func retrying(cmd string, retries int, delay time.Duration) PlannedStep {
	d := config.Duration(delay)
	return PlannedStep{Step: Step{Name: cmd, Retries: retries, RetryDelay: &d}, Command: cmd}
}

func TestRetriesUntilSuccess(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"x": {executor.StatusFailed, executor.StatusFailed, executor.StatusOK}}}
	r := (&Engine{Exec: ex, Steps: []PlannedStep{retrying("x", 5, time.Millisecond)}}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusOK || r.Steps[0].Attempts != 3 {
		t.Errorf("result = %+v", r)
	}
}

func TestRetriesAreLimited(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"x": {executor.StatusFailed}}}
	r := (&Engine{Exec: ex, Steps: []PlannedStep{retrying("x", 2, time.Millisecond)}}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusFailed || len(ex.calls) != 3 {
		t.Errorf("status = %s, calls = %d", r.Status, len(ex.calls))
	}
}

func TestCancelStopsRetryWait(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"x": {executor.StatusFailed}}}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	(&Engine{Exec: ex, Steps: []PlannedStep{retrying("x", 5, time.Hour)}}).Run(ctx, srv, executor.Request{})
	if time.Since(start) > 2*time.Second {
		t.Error("the retry delay ignored cancellation")
	}
	if len(ex.calls) != 1 {
		t.Errorf("calls = %d", len(ex.calls))
	}
}

func TestCanceledStepIsNotRetried(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"x": {executor.StatusCanceled}}}
	(&Engine{Exec: ex, Steps: []PlannedStep{retrying("x", 5, time.Millisecond)}}).Run(context.Background(), srv, executor.Request{})
	if len(ex.calls) != 1 {
		t.Errorf("calls = %d", len(ex.calls))
	}
}

func TestContinueOnError(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"opt": {executor.StatusFailed}}}
	steps := plan("opt", "next")
	steps[0].Step.ContinueOnError = true
	r := (&Engine{Exec: ex, Steps: steps}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusOK || len(ex.calls) != 2 {
		t.Errorf("result = %+v, calls = %v", r, ex.calls)
	}
	if r.Steps[0].Status != executor.StatusFailed {
		t.Errorf("the failure must stay visible: %+v", r.Steps[0])
	}
}

func TestContinueOnErrorDoesNotIgnoreCancel(t *testing.T) {
	ex := &scriptedExec{answers: map[string][]executor.Status{"opt": {executor.StatusCanceled}}}
	steps := plan("opt", "next")
	steps[0].Step.ContinueOnError = true
	r := (&Engine{Exec: ex, Steps: steps}).Run(context.Background(), srv, executor.Request{})
	if r.Status != executor.StatusCanceled || len(ex.calls) != 1 {
		t.Errorf("result = %+v, calls = %v", r, ex.calls)
	}
}

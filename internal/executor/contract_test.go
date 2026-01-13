package executor_test

// This is an external test package (executor_test): it can only use the
// exported API, exactly like real callers.

import (
	"context"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
	"example.com/msh/internal/inventory"
)

// contractCase is behavior every Executor must show.
type contractCase struct {
	name       string
	req        executor.Request
	ctx        func() (context.Context, context.CancelFunc)
	wantStatus executor.Status
}

func contractCases() []contractCase {
	background := func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }
	canceled := func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, func() {}
	}
	return []contractCase{
		{"success", executor.Request{Command: "true"}, background, executor.StatusOK},
		{"canceled before start", executor.Request{Command: "sleep 5"}, canceled, executor.StatusCanceled},
	}
}

func TestExecutorContract(t *testing.T) {
	executors := map[string]executor.Executor{
		"local": executor.LocalExecutor{},
		"fake":  &fake.Executor{},
	}
	for name, ex := range executors {
		for _, tc := range contractCases() {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := tc.ctx()
				defer cancel()
				start := time.Now()
				r := ex.Run(ctx, inventory.Server{Name: "s1", Address: "192.0.2.1"}, tc.req)
				if r.Status != tc.wantStatus {
					t.Errorf("status = %s, want %s (err %v)", r.Status, tc.wantStatus, r.Err)
				}
				if r.Server != "s1" {
					t.Errorf("server = %q", r.Server)
				}
				if time.Since(start) > 3*time.Second {
					t.Error("Run did not return promptly")
				}
			})
		}
	}
}

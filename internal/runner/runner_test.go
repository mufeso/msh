package runner

import (
	"context"
	"strings"
	"testing"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
	"example.com/msh/internal/inventory"
)

func servers(names ...string) []inventory.Server {
	out := make([]inventory.Server, len(names))
	for i, n := range names {
		out[i] = inventory.Server{Name: n, Address: "192.0.2." + n}
	}
	return out
}

func TestRunSequential(t *testing.T) {
	f := &fake.Executor{}
	var emitted []string
	results := Run(context.Background(), f, servers("a", "b", "c"), executor.Request{Command: "uptime"}, func(r executor.Result) {
		emitted = append(emitted, r.Server)
	})
	if len(results) != 3 || results[2].Server != "c" {
		t.Errorf("results = %+v", results)
	}
	if len(emitted) != 3 || emitted[0] != "a" {
		t.Errorf("emitted = %v", emitted)
	}
	if calls := f.Calls(); len(calls) != 3 || calls[1].Request.Command != "uptime" {
		t.Errorf("calls = %+v", calls)
	}
}

func TestRunStopsSchedulingAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fake.Executor{}
	var seen int
	results := Run(ctx, f, servers("a", "b", "c", "d"), executor.Request{}, func(executor.Result) {
		seen++
		if seen == 2 {
			cancel() // like CTRL+C after the second server
		}
	})
	if len(f.Calls()) != 2 {
		t.Errorf("executor called %d times, want 2", len(f.Calls()))
	}
	if len(results) != 4 {
		t.Fatalf("results = %d, want 4 (every server reported)", len(results))
	}
	for _, r := range results[2:] {
		if r.Status != executor.StatusCanceled || !strings.Contains(r.Err.Error(), "not started") {
			t.Errorf("result = %+v", r)
		}
	}
}

func TestNotStartedGlobalTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	results := Run(ctx, &fake.Executor{}, servers("a"), executor.Request{}, nil)
	if results[0].Status != executor.StatusTimeout || errs.KindOf(results[0].Err) != errs.KindGlobalTimeout {
		t.Errorf("result = %+v", results[0])
	}
}

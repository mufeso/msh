package runner

import (
	"context"
	"strings"
	"testing"
	"time"

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
	results := Run(context.Background(), f, servers("a", "b", "c"), executor.Request{Command: "uptime"}, Options{}, func(r executor.Result) {
		emitted = append(emitted, r.Server)
	}).Results
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
	results := Run(ctx, f, servers("a", "b", "c", "d"), executor.Request{}, Options{}, func(executor.Result) {
		seen++
		if seen == 2 {
			cancel() // like CTRL+C after the second server
		}
	}).Results
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
	results := Run(ctx, &fake.Executor{}, servers("a"), executor.Request{}, Options{}, nil).Results
	if results[0].Status != executor.StatusTimeout || errs.KindOf(results[0].Err) != errs.KindGlobalTimeout {
		t.Errorf("result = %+v", results[0])
	}
}

func TestFailFast(t *testing.T) {
	f := &fake.Executor{Responses: map[string]fake.Response{
		"b": {Result: executor.Result{Status: executor.StatusFailed, ExitCode: 1}},
	}}
	report := Run(context.Background(), f, servers("a", "b", "c"), executor.Request{}, Options{FailFast: true}, nil)
	if len(f.Calls()) != 2 {
		t.Errorf("calls = %d, want 2", len(f.Calls()))
	}
	if report.Results[2].Status != executor.StatusCanceled {
		t.Errorf("c = %+v", report.Results[2])
	}

	f = &fake.Executor{Responses: f.Responses}
	report = Run(context.Background(), f, servers("a", "b", "c"), executor.Request{}, Options{}, nil)
	if len(f.Calls()) != 3 || report.Summary.Failed != 1 || report.Summary.OK != 2 {
		t.Errorf("without fail-fast: calls = %d, summary = %+v", len(f.Calls()), report.Summary)
	}
}

func TestTimeoutIsPerServer(t *testing.T) {
	f := &fake.Executor{Responses: map[string]fake.Response{"slow": {Delay: time.Hour}}}
	report := Run(context.Background(), f, servers("a", "slow", "b"), executor.Request{Timeout: 50 * time.Millisecond}, Options{}, nil)
	got := map[string]executor.Status{}
	for _, r := range report.Results {
		got[r.Server] = r.Status
	}
	if got["a"] != executor.StatusOK || got["slow"] != executor.StatusTimeout || got["b"] != executor.StatusOK {
		t.Errorf("statuses = %v", got)
	}
}

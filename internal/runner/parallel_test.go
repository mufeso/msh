package runner

import (
	"context"
	"runtime"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
)

func TestParallelRunsConcurrently(t *testing.T) {
	f := &fake.Executor{Default: fake.Response{Delay: 100 * time.Millisecond}}
	start := time.Now()
	var emitted int
	report := Run(context.Background(), f, servers("a", "b", "c", "d"), executor.Request{}, Options{Parallel: true, Workers: 4}, func(executor.Result) {
		emitted++ // safe: emit calls never overlap
	})
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		t.Errorf("4 x 100ms took %v; not parallel", elapsed)
	}
	if emitted != 4 || report.Summary.OK != 4 {
		t.Errorf("emitted = %d, summary = %+v", emitted, report.Summary)
	}
	for i, name := range []string{"a", "b", "c", "d"} {
		if report.Results[i].Server != name {
			t.Errorf("results[%d] = %s, want %s (selection order)", i, report.Results[i].Server, name)
		}
	}
}

func TestPoolBoundsConcurrency(t *testing.T) {
	f := &fake.Executor{Default: fake.Response{Delay: 30 * time.Millisecond}}
	names := make([]string, 20)
	for i := range names {
		names[i] = string(rune('a' + i))
	}
	report := Run(context.Background(), f, servers(names...), executor.Request{}, Options{Parallel: true, Workers: 3}, nil)
	if got := f.MaxConcurrent(); got > 3 {
		t.Errorf("max concurrent = %d, want <= 3", got)
	}
	if report.Summary.OK != 20 {
		t.Errorf("summary = %+v", report.Summary)
	}
}

func TestWorkerCount(t *testing.T) {
	tests := []struct{ configured, servers, want int }{
		{5, 3, 3}, {5, 10, 5}, {0, 10, 1}, {-2, 10, 1}, {5, 0, 1},
	}
	for _, tt := range tests {
		if got := workerCount(tt.configured, tt.servers); got != tt.want {
			t.Errorf("workerCount(%d, %d) = %d, want %d", tt.configured, tt.servers, got, tt.want)
		}
	}
}

func TestExactConcurrency(t *testing.T) {
	f := &fake.Executor{Default: fake.Response{Delay: 50 * time.Millisecond}}
	Run(context.Background(), f, servers("a", "b", "c", "d", "e", "f"), executor.Request{}, Options{Parallel: true, Workers: 2}, nil)
	if got := f.MaxConcurrent(); got != 2 {
		t.Errorf("max concurrent = %d, want exactly 2", got)
	}
}

func TestEmptySelection(t *testing.T) {
	report := Run(context.Background(), &fake.Executor{}, nil, executor.Request{}, Options{Parallel: true, Workers: 4}, nil)
	if report.Summary.Total != 0 || len(report.Results) != 0 {
		t.Errorf("report = %+v", report)
	}
}

func TestNoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		Run(context.Background(), &fake.Executor{}, servers("a", "b", "c"), executor.Request{}, Options{Parallel: true, Workers: 2}, nil)
	}
	deadline := time.Now().Add(time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines before %d, after %d", before, after)
	}
}

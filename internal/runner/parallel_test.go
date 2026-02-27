package runner

import (
	"context"
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

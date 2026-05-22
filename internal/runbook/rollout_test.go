package runbook

import (
	"context"
	"sync"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// timeline records when servers start and end.
type timeline struct {
	mu     sync.Mutex
	events []string
	fail   map[string]bool
	delay  time.Duration
}

func (l *timeline) Run(_ context.Context, s inventory.Server, _ executor.Request) executor.Result {
	l.add("start " + s.Name)
	time.Sleep(l.delay)
	l.add("end " + s.Name)
	r := executor.Result{Server: s.Name, Status: executor.StatusOK}
	if l.fail[s.Name] {
		r.Status, r.ExitCode = executor.StatusFailed, 1
	}
	return r
}

func (l *timeline) add(e string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, e)
}

func (l *timeline) index(e string) int {
	for i, v := range l.events {
		if v == e {
			return i
		}
	}
	return -1
}

func TestRolloutWaitsForEachBatch(t *testing.T) {
	l := &timeline{delay: 10 * time.Millisecond}
	report := Rollout{BatchSize: 2}.Run(context.Background(), l, fleet(4), runner.Options{Workers: 4}, nil)
	if report.Summary.OK != 4 {
		t.Fatalf("summary = %+v", report.Summary)
	}
	for _, first := range []string{"end s1", "end s2"} {
		for _, second := range []string{"start s3", "start s4"} {
			if l.index(first) > l.index(second) {
				t.Errorf("%q came after %q: %v", first, second, l.events)
			}
		}
	}
	for i, r := range report.Results {
		if r.Server != fleet(4)[i].Name {
			t.Errorf("results out of order: %+v", report.Results)
		}
	}
}

func TestRolloutRespectsWorkersInsideABatch(t *testing.T) {
	f := &fake.Executor{Default: fake.Response{Delay: 20 * time.Millisecond}}
	Rollout{BatchSize: 3}.Run(context.Background(), f, fleet(6), runner.Options{Workers: 10}, nil)
	if got := f.MaxConcurrent(); got != 3 {
		t.Errorf("max concurrent = %d, want the batch size 3", got)
	}
}

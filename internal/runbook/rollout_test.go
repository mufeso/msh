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

func TestRolloutStopsAfterFailure(t *testing.T) {
	l := &timeline{fail: map[string]bool{"s2": true}}
	report := Rollout{BatchSize: 2, MaxFailures: 0}.Run(context.Background(), l, fleet(6), runner.Options{Workers: 2}, nil)
	s := report.Summary
	if s.Total != 6 || s.OK != 1 || s.Failed != 1 || s.Skipped != 4 {
		t.Fatalf("summary = %+v", s)
	}
	if l.index("start s3") != -1 {
		t.Error("the next batch started after the limit was exceeded")
	}
	if got := report.Results[5]; got.Status != executor.StatusSkipped || got.Err == nil {
		t.Errorf("s6 = %+v", got)
	}
}

func TestRolloutToleratesFailuresUpToTheLimit(t *testing.T) {
	l := &timeline{fail: map[string]bool{"s1": true, "s3": true}}
	report := Rollout{BatchSize: 1, MaxFailures: 1}.Run(context.Background(), l, fleet(4), runner.Options{Workers: 1}, nil)
	s := report.Summary
	if s.Failed != 2 || s.OK != 1 || s.Skipped != 1 {
		t.Errorf("summary = %+v", s)
	}
}

func TestNewRolloutDefaults(t *testing.T) {
	r, _ := (&Runbook{}).NewRollout(10)
	if r.BatchSize != 0 || r.MaxFailures != -1 {
		t.Errorf("no settings: %+v", r)
	}
	r, _ = (&Runbook{Batch: "25%"}).NewRollout(10)
	if r.BatchSize != 3 || r.MaxFailures != 0 {
		t.Errorf("batch: %+v", r)
	}
	r, _ = (&Runbook{Batch: "2", MaxFailures: "10%"}).NewRollout(25)
	if r.MaxFailures != 3 {
		t.Errorf("percent failures: %+v", r)
	}
}

func TestPauseBetweenBatches(t *testing.T) {
	l := &timeline{}
	start := time.Now()
	Rollout{BatchSize: 1, MaxFailures: -1, Pause: 30 * time.Millisecond}.Run(context.Background(), l, fleet(3), runner.Options{Workers: 1}, nil)
	if elapsed := time.Since(start); elapsed < 60*time.Millisecond {
		t.Errorf("two pauses took only %v", elapsed)
	}
}

func TestCancelDuringPause(t *testing.T) {
	l := &timeline{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	report := Rollout{BatchSize: 1, MaxFailures: -1, Pause: time.Hour}.Run(ctx, l, fleet(3), runner.Options{Workers: 1}, nil)
	if time.Since(start) > 2*time.Second {
		t.Error("the pause ignored cancellation")
	}
	if report.Summary.OK != 1 || report.Summary.Skipped != 2 {
		t.Errorf("summary = %+v", report.Summary)
	}
}

func TestCanaryGroups(t *testing.T) {
	tests := []struct {
		r    Rollout
		n    int
		want string
	}{
		{Rollout{Canary: 1, BatchSize: 2}, 5, "s1|s2,s3|s4,s5"},
		{Rollout{Canary: 1}, 4, "s1|s2,s3,s4"},
		{Rollout{Canary: 9, BatchSize: 2}, 3, "s1,s2|s3"},
		{Rollout{BatchSize: 2}, 3, "s1,s2|s3"},
	}
	for _, tt := range tests {
		if got := shape(tt.r.Groups(fleet(tt.n))); got != tt.want {
			t.Errorf("%+v: %q, want %q", tt.r, got, tt.want)
		}
	}
}

func TestCanaryFailureProtectsTheFleet(t *testing.T) {
	l := &timeline{fail: map[string]bool{"s1": true}}
	rb := Runbook{Canary: 1}
	r, _ := rb.NewRollout(4)
	report := r.Run(context.Background(), l, fleet(4), runner.Options{Workers: 4}, nil)
	if report.Summary.Failed != 1 || report.Summary.Skipped != 3 {
		t.Errorf("summary = %+v", report.Summary)
	}
}

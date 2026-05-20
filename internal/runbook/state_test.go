package runbook

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"example.com/msh/internal/executor"
)

func TestTrackerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")
	tr, err := NewTracker(path, "r", "h1", false)
	if err != nil {
		t.Fatal(err)
	}
	tr.Mark("s1", 2)
	if tr.Err() != nil {
		t.Fatal(tr.Err())
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", info.Mode().Perm())
	}
	again, err := NewTracker(path, "r", "h1", true)
	if err != nil {
		t.Fatal(err)
	}
	if again.Completed("s1") != 2 || again.Completed("s2") != 0 {
		t.Errorf("completed = %d", again.Completed("s1"))
	}
	if _, err := NewTracker(path, "r", "other", true); err == nil {
		t.Error("a changed plan was accepted")
	}
	if err := again.Finish(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTracker(path, "r", "h1", true); !errors.Is(err, ErrNoState) {
		t.Errorf("err = %v", err)
	}
}

func TestTrackerConcurrentMarks(t *testing.T) {
	tr, _ := NewTracker(filepath.Join(t.TempDir(), "s.json"), "r", "h", false)
	var wg sync.WaitGroup
	for _, s := range []string{"a", "b", "c", "d"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 1; i <= 5; i++ {
				tr.Mark(s, i)
			}
		}()
	}
	wg.Wait()
	if tr.Err() != nil || tr.Completed("c") != 5 {
		t.Errorf("err = %v, completed = %d", tr.Err(), tr.Completed("c"))
	}
}

func TestPlanHashChangesWithCommands(t *testing.T) {
	if PlanHash(plan("a", "b")) == PlanHash(plan("a", "c")) {
		t.Error("different plans have the same hash")
	}
	if PlanHash(plan("a")) != PlanHash(plan("a")) {
		t.Error("the hash is not stable")
	}
}

type memProgress map[string]int

func (m memProgress) Completed(s string) int { return m[s] }
func (m memProgress) Mark(s string, n int)   { m[s] = n }

func TestEngineResumes(t *testing.T) {
	ex := &scriptedExec{}
	p := memProgress{"s1": 2}
	r := (&Engine{Exec: ex, Steps: plan("a", "b", "c"), Progress: p}).Run(context.Background(), srv, executor.Request{})
	if len(ex.calls) != 1 || ex.calls[0] != "c" {
		t.Errorf("calls = %v, want only c", ex.calls)
	}
	if r.Status != executor.StatusOK || r.Steps[0].Status != executor.StatusSkipped || p["s1"] != 3 {
		t.Errorf("result = %+v, progress = %v", r, p)
	}
}

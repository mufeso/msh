package runbook

import (
	"context"
	"sync"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

type startRecorder struct {
	mu     sync.Mutex
	starts []time.Time
}

func (r *startRecorder) Run(ctx context.Context, s inventory.Server, _ executor.Request) executor.Result {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts = append(r.starts, time.Now())
	return executor.Result{Server: s.Name, Status: executor.StatusOK}
}

func TestStaggerSpacesStarts(t *testing.T) {
	rec := &startRecorder{}
	s := &Stagger{Exec: rec, Interval: 40 * time.Millisecond}
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Run(context.Background(), srv, executor.Request{})
		}()
	}
	wg.Wait()
	if got := rec.starts[2].Sub(rec.starts[0]); got < 70*time.Millisecond {
		t.Errorf("three starts within %v", got)
	}
}

func TestStaggerStopsOnCancel(t *testing.T) {
	s := &Stagger{Exec: &startRecorder{}, Interval: time.Hour}
	s.reserve() // the next slot is an hour away
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	s.Run(ctx, srv, executor.Request{})
	if time.Since(start) > 2*time.Second {
		t.Error("the wait ignored cancellation")
	}
}

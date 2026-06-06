package runbook

import (
	"context"
	"sync"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Stagger is an Executor that spaces out the start of each run by
// Interval. Each caller reserves the next free time slot under the lock
// and sleeps outside it, so waiting callers do not block each other.
type Stagger struct {
	Exec     executor.Executor
	Interval time.Duration

	mu   sync.Mutex
	next time.Time
}

// Run waits for a slot and then runs. If ctx ends while waiting, the
// wrapped executor reports the cancellation.
func (s *Stagger) Run(ctx context.Context, server inventory.Server, req executor.Request) executor.Result {
	wait(ctx, time.Until(s.reserve()))
	return s.Exec.Run(ctx, server, req)
}

// reserve returns the next free slot and moves it forward.
func (s *Stagger) reserve() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.next.Before(now) {
		s.next = now
	}
	slot := s.next
	s.next = slot.Add(s.Interval)
	return slot
}

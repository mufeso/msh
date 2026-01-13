package fake

import (
	"context"
	"sync"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

func TestCannedResponses(t *testing.T) {
	f := &Executor{Responses: map[string]Response{
		"db01": {Result: executor.Result{Status: executor.StatusFailed, ExitCode: 2, Stderr: "boom"}},
	}}
	ok := f.Run(context.Background(), inventory.Server{Name: "web01"}, executor.Request{Command: "x"})
	bad := f.Run(context.Background(), inventory.Server{Name: "db01"}, executor.Request{Command: "y"})
	if ok.Status != executor.StatusOK || bad.Status != executor.StatusFailed || bad.ExitCode != 2 {
		t.Errorf("ok = %+v, bad = %+v", ok, bad)
	}
	calls := f.Calls()
	if len(calls) != 2 || calls[1].Server != "db01" || calls[1].Request.Command != "y" {
		t.Errorf("calls = %+v", calls)
	}
}

func TestDelayHonorsCancellation(t *testing.T) {
	f := &Executor{Default: Response{Delay: time.Hour}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	r := f.Run(ctx, inventory.Server{Name: "slow"}, executor.Request{})
	if r.Status != executor.StatusTimeout {
		t.Errorf("status = %s, want timeout", r.Status)
	}
}

func TestMaxConcurrent(t *testing.T) {
	f := &Executor{Default: Response{Delay: 50 * time.Millisecond}}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Run(context.Background(), inventory.Server{Name: "s"}, executor.Request{})
		}()
	}
	wg.Wait()
	if got := f.MaxConcurrent(); got != 3 {
		t.Errorf("MaxConcurrent = %d, want 3", got)
	}
}

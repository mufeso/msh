// Package fake provides a configurable Executor for tests.
//
// It never touches the network or starts processes. Each server can be
// given a canned Result and a delay; the delay honors context
// cancellation exactly like a real executor must.
package fake

import (
	"context"
	"errors"
	"sync"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Response is the canned behavior for one server.
type Response struct {
	Result executor.Result // Server, Address, Started and Duration are filled in
	Delay  time.Duration   // how long Run "takes"
}

// Call records one invocation of Run.
type Call struct {
	Server  string
	Request executor.Request
}

// Executor is a fake executor.Executor. The zero value answers every
// server with StatusOK immediately. It is safe for concurrent use.
type Executor struct {
	// Responses maps server names to behaviors; Default is used for
	// servers that are not in the map. Set them before calling Run.
	Responses map[string]Response
	Default   Response

	mu         sync.Mutex // protects the fields below
	calls      []Call
	running    int
	maxRunning int
}

// Compile-time check that *Executor implements executor.Executor.
var _ executor.Executor = (*Executor)(nil)

// Run implements executor.Executor.
func (f *Executor) Run(ctx context.Context, server inventory.Server, req executor.Request) executor.Result {
	f.begin(server.Name, req)
	defer f.end()

	resp, ok := f.Responses[server.Name]
	if !ok {
		resp = f.Default
	}
	started := time.Now()

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	if resp.Delay > 0 {
		// NewTimer + Stop instead of time.After: the timer is released as
		// soon as Run returns, even if ctx ends first.
		timer := time.NewTimer(resp.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return interrupted(ctx.Err(), server, started)
		}
	} else if err := ctx.Err(); err != nil {
		return interrupted(err, server, started)
	}

	r := resp.Result
	r.Server, r.Address = server.Name, server.Address
	r.Started, r.Duration = started, time.Since(started)
	if r.Status == "" {
		r.Status = executor.StatusOK
	}
	return r
}

// interrupted builds the result for a context that ended early.
func interrupted(err error, server inventory.Server, started time.Time) executor.Result {
	st := executor.StatusCanceled
	if errors.Is(err, context.DeadlineExceeded) {
		st = executor.StatusTimeout
	}
	return executor.Result{
		Server: server.Name, Address: server.Address, Status: st, ExitCode: -1,
		Err: err, Started: started, Duration: time.Since(started),
	}
}

func (f *Executor) begin(server string, req executor.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Call{Server: server, Request: req})
	f.running++
	if f.running > f.maxRunning {
		f.maxRunning = f.running
	}
}

func (f *Executor) end() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running--
}

// Calls returns a copy of the recorded calls, in call order.
func (f *Executor) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Call(nil), f.calls...)
}

// MaxConcurrent returns the largest number of simultaneous Run calls seen.
func (f *Executor) MaxConcurrent() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxRunning
}

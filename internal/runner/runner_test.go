package runner

import (
	"context"
	"testing"

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
	results := Run(context.Background(), f, servers("a", "b", "c"), executor.Request{Command: "uptime"}, func(r executor.Result) {
		emitted = append(emitted, r.Server)
	})
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

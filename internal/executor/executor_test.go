package executor

import (
	"context"
	"testing"

	"example.com/msh/internal/inventory"
)

// echoExecutor is the smallest possible Executor: it shows that any type
// with a matching Run method satisfies the interface.
type echoExecutor struct{}

func (echoExecutor) Run(_ context.Context, s inventory.Server, req Request) Result {
	return Result{Server: s.Name, Status: StatusOK, Stdout: req.Command}
}

func TestInterfaceSatisfiedImplicitly(t *testing.T) {
	var e Executor = echoExecutor{}
	r := e.Run(context.Background(), inventory.Server{Name: "web01"}, Request{Command: "hello"})
	if r.Server != "web01" || r.Stdout != "hello" || r.Status != StatusOK {
		t.Errorf("result = %+v", r)
	}
}

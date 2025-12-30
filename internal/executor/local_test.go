package executor

import (
	"context"
	"testing"

	"example.com/msh/internal/inventory"
)

var localServer = inventory.Server{Name: "local", Address: "localhost"}

func TestLocalExecutorStdout(t *testing.T) {
	r := LocalExecutor{}.Run(context.Background(), localServer, Request{Command: "echo hello"})
	if r.Status != StatusOK || r.Stdout != "hello\n" || r.ExitCode != 0 {
		t.Errorf("result = %+v", r)
	}
	if r.Duration <= 0 {
		t.Error("duration not recorded")
	}
}

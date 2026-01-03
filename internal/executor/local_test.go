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

func TestLocalExecutorOutputLimit(t *testing.T) {
	r := LocalExecutor{}.Run(context.Background(), localServer, Request{Command: "printf 0123456789", OutputLimit: 4})
	if r.Stdout != "0123" || !r.StdoutTruncated || r.Status != StatusOK {
		t.Errorf("result = %+v", r)
	}
}

func TestLocalExecutorSeparatesStreams(t *testing.T) {
	r := LocalExecutor{}.Run(context.Background(), localServer, Request{Command: "echo out; echo err >&2"})
	if r.Stdout != "out\n" || r.Stderr != "err\n" {
		t.Errorf("stdout = %q, stderr = %q", r.Stdout, r.Stderr)
	}
}

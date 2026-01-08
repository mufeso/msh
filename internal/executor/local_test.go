package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestLocalExecutorExitCodes(t *testing.T) {
	tests := []struct {
		command    string
		wantStatus Status
		wantCode   int
	}{
		{"true", StatusOK, 0},
		{"exit 3", StatusFailed, 3},
		{"false", StatusFailed, 1},
		{"command-that-does-not-exist-xyz", StatusFailed, 127},
	}
	for _, tt := range tests {
		r := LocalExecutor{}.Run(context.Background(), localServer, Request{Command: tt.command})
		if r.Status != tt.wantStatus || r.ExitCode != tt.wantCode {
			t.Errorf("%q: status=%s exit=%d, want %s %d", tt.command, r.Status, r.ExitCode, tt.wantStatus, tt.wantCode)
		}
		if r.Err != nil {
			t.Errorf("%q: Err = %v, want nil for a command that ran", tt.command, r.Err)
		}
	}
}

func TestLocalExecutorTimeout(t *testing.T) {
	start := time.Now()
	r := LocalExecutor{}.Run(context.Background(), localServer, Request{Command: "sleep 5", Timeout: 100 * time.Millisecond})
	if r.Status != StatusTimeout {
		t.Errorf("status = %s, err = %v, want timeout", r.Status, r.Err)
	}
	if !errors.Is(r.Err, context.DeadlineExceeded) {
		t.Errorf("err = %v", r.Err)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("timeout did not stop the command quickly")
	}
}

func TestLocalExecutorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled
	r := LocalExecutor{}.Run(ctx, localServer, Request{Command: "echo never"})
	if r.Status != StatusCanceled {
		t.Errorf("status = %s, want canceled", r.Status)
	}
}

func TestLocalExecutorKillsWholeProcessGroup(t *testing.T) {
	// The shell is not the only process here: sleep is its child. Without
	// a process group, Run would wait for sleep to finish.
	start := time.Now()
	r := LocalExecutor{}.Run(context.Background(), localServer, Request{
		Command: "sleep 30; echo finished",
		Timeout: 200 * time.Millisecond,
	})
	if r.Status != StatusTimeout {
		t.Errorf("status = %s", r.Status)
	}
	if strings.Contains(r.Stdout, "finished") {
		t.Error("command was not stopped")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Run took %v; the child process kept it waiting", elapsed)
	}
}

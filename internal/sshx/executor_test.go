package sshx

import (
	"context"
	"net"
	"runtime"
	"strconv"
	"testing"
	"time"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshtest"
)

func (e *testEnv) server0() inventory.Server {
	host, portStr, _ := net.SplitHostPort(e.server.Addr)
	port, _ := strconv.Atoi(portStr)
	return inventory.Server{Name: "s1", Address: host, Port: port, User: "tester"}
}

func TestExecutorRun(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{})
	ex := &Executor{Connector: env.dialer(t)}
	r := ex.Run(context.Background(), env.server0(), executor.Request{Command: "echo out; echo err >&2; exit 5"})
	if r.Status != executor.StatusFailed || r.ExitCode != 5 || r.Stdout != "out\n" || r.Stderr != "err\n" {
		t.Errorf("result = %+v", r)
	}
	if !env.server.WaitIdle(2e9) {
		t.Error("session left open on the server")
	}
}

func TestExecutorStdin(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{})
	ex := &Executor{Connector: env.dialer(t)}
	r := ex.Run(context.Background(), env.server0(), executor.Request{Command: "sh -s", Stdin: []byte("echo from-stdin\n")})
	if r.Status != executor.StatusOK || r.Stdout != "from-stdin\n" {
		t.Errorf("result = %+v", r)
	}
}

func TestExecutorConnectionFailure(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{})
	env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), "")
	ex := &Executor{Connector: env.dialer(t)}
	r := ex.Run(context.Background(), env.server0(), executor.Request{Command: "true"})
	if r.Status != executor.StatusError || errs.KindOf(r.Err) != errs.KindHostKey {
		t.Errorf("result = %+v", r)
	}
	if len(env.server.Commands()) != 0 {
		t.Error("command ran despite host key failure")
	}
}

func TestExecutorCommandTimeout(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{Handler: sshtest.Blocking("started\n")})
	ex := &Executor{Connector: env.dialer(t)}
	start := time.Now()
	r := ex.Run(context.Background(), env.server0(), executor.Request{Command: "wait", Timeout: 200 * time.Millisecond})
	if r.Status != executor.StatusTimeout || errs.KindOf(r.Err) != errs.KindRemoteTimeout {
		t.Errorf("result = %+v", r)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("timeout was not enforced promptly")
	}
	if !env.server.WaitIdle(2 * time.Second) {
		t.Error("remote session still open after timeout")
	}
	if sig := env.server.Signals(); len(sig) == 0 || sig[0] != "TERM" {
		t.Errorf("server signals = %v, want TERM", sig)
	}
}

func TestExecutorCancellation(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{Handler: sshtest.Blocking("")})
	ex := &Executor{Connector: env.dialer(t)}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel() // like CTRL+C
	}()
	r := ex.Run(ctx, env.server0(), executor.Request{Command: "wait"})
	if r.Status != executor.StatusCanceled || errs.KindOf(r.Err) != errs.KindCanceled {
		t.Errorf("result = %+v", r)
	}
	if !env.server.WaitIdle(2 * time.Second) {
		t.Error("remote session still open after cancellation")
	}
}

func TestExecutorGlobalTimeout(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{Handler: sshtest.Blocking("")})
	ex := &Executor{Connector: env.dialer(t)}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	r := ex.Run(ctx, env.server0(), executor.Request{Command: "wait", Timeout: time.Hour})
	if r.Status != executor.StatusTimeout || errs.KindOf(r.Err) != errs.KindGlobalTimeout {
		t.Errorf("result = %+v", r)
	}
}

func TestExecutorNoGoroutineLeak(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{Handler: sshtest.Blocking("")})
	ex := &Executor{Connector: env.dialer(t)}
	before := runtime.NumGoroutine()
	for i := 0; i < 5; i++ {
		ex.Run(context.Background(), env.server0(), executor.Request{Command: "wait", Timeout: 50 * time.Millisecond})
	}
	env.server.WaitIdle(2 * time.Second)
	// Give finished goroutines a moment to exit, then compare.
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Errorf("goroutines: before %d, after %d", before, after)
	}
}

func TestExecutorCommandIgnoringStdinSucceeds(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{Handler: sshtest.Static("done\n", "", 0)})
	ex := &Executor{Connector: env.dialer(t)}
	stdin := make([]byte, 4<<20) // more than the SSH window: writing must fail
	for i := 0; i < 3; i++ {
		r := ex.Run(context.Background(), env.server0(), executor.Request{Command: "x", Stdin: stdin})
		if r.Status != executor.StatusOK || r.Stdout != "done\n" {
			t.Fatalf("result = %+v", r)
		}
	}
}

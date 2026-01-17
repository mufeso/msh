package sshx

import (
	"context"
	"net"
	"strconv"
	"testing"

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

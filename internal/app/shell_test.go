package app

import (
	"context"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestShellRunsRemoteShell(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	ctx := withInput(context.Background(), "echo from-remote-shell\nexit 7\n")
	args := append([]string{"shell", "--host", "s1"}, fleet.args()...)
	stdout, stderr, code := runCLICtx(t, ctx, nil, args...)
	if code != 7 {
		t.Errorf("code = %v, want the remote exit status 7 (stderr %q)", code, stderr)
	}
	if !contains(stdout, "from-remote-shell") {
		t.Errorf("stdout = %q", stdout)
	}
	ptys := fleet.servers[0].PTYs()
	if len(ptys) != 1 || ptys[0].Term != "xterm-256color" {
		t.Errorf("pty requests = %+v", ptys)
	}
}

func TestShellNeedsExactlyOneServer(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	_, stderr, code := fleet.run(t, []string{"shell"}, "--all")
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "exactly one") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestShellCommandRunsRemotely(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	args := append([]string{"shell", "--host", "s1", "--command", "echo interactive-cmd; exit 4"}, fleet.args()...)
	stdout, _, code := runCLICtx(t, context.Background(), nil, args...)
	if code != 4 || !contains(stdout, "interactive-cmd") {
		t.Errorf("code = %v, stdout = %q", code, stdout)
	}
	if cmds := fleet.servers[0].Commands(); len(cmds) != 1 || cmds[0] != "echo interactive-cmd; exit 4" {
		t.Errorf("remote commands = %q", cmds)
	}
	if len(fleet.servers[0].PTYs()) != 1 {
		t.Error("no pseudo-terminal requested for the interactive command")
	}
}

func TestInteractiveTask(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, `  message:
    description: Follow a log filtered by a pattern
    interpreter: sh
    command: "printf 'boot\nsshd: accepted\nkernel\nsshd: closed\n' | grep --line-buffered -- {{pattern}}"
    arguments:
      - name: pattern
        required: true
    interactive: true
`)
	args := append([]string{"c", "message", "s1", "sshd", "--config", cfg}, fleet.args()...)
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "sshd: accepted") || contains(stdout, "kernel") {
		t.Errorf("stdout = %q", stdout)
	}
	if len(fleet.servers[0].PTYs()) != 1 {
		t.Error("interactive task did not request a PTY")
	}
}

func TestShellForwardsTermAndSize(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	args := append([]string{"shell", "--host", "s1", "--command", "exit 0"}, fleet.args()...)
	if _, stderr, code := runCLICtx(t, context.Background(), map[string]string{"TERM": "screen-256color"}, args...); code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	p := fleet.servers[0].PTYs()
	if len(p) != 1 || p[0].Term != "screen-256color" || p[0].Width != 100 || p[0].Height != 30 {
		t.Errorf("pty = %+v", p)
	}
}

func TestShellThroughBastion(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	args := append([]string{"shell", "--host", "s1", "--command", "echo hello-via-jump"}, fleet.args()...)
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || !contains(stdout, "hello-via-jump") {
		t.Errorf("code = %v, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}

func TestShellConnectionError(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	fleet.servers[0].Close()
	args := append([]string{"shell", "--host", "s1"}, fleet.args()...)
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.Connection || !contains(stderr, "TCP connection failed") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

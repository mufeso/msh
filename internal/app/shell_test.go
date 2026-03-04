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

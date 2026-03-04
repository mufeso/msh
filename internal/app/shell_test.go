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

package app

import (
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
)

func TestShortcutsSendTheirCommand(t *testing.T) {
	for _, sc := range shortcuts {
		t.Run(sc.name, func(t *testing.T) {
			fleet := newTestFleet(t, 1, sshtest.Static("out\n", "", 0))
			_, stderr, code := fleet.run(t, []string{sc.name}, "--host", "s1")
			if exitcode.Code(code) != exitcode.OK {
				t.Fatalf("code = %d, stderr = %s", code, stderr)
			}
			if cmds := fleet.servers[0].Commands(); len(cmds) != 1 || cmds[0] != sc.command {
				t.Errorf("remote commands = %q, want %q", cmds, sc.command)
			}
		})
	}
}

func TestPingAndHostsCheck(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	stdout, _, code := fleet.run(t, []string{"ping"}, "--all")
	if exitcode.Code(code) != exitcode.OK || !contains(stdout, "=== s1: ok") {
		t.Errorf("ping: code = %d\n%s", code, stdout)
	}
	fleet.servers[1].Close()
	stdout, _, code = fleet.run(t, []string{"hosts", "check"})
	if exitcode.Code(code) != exitcode.Connection || !contains(stdout, "=== s2: error") {
		t.Errorf("hosts check: code = %d\n%s", code, stdout)
	}
}

func TestRealShortcuts(t *testing.T) {
	fleet := newTestFleet(t, 1, nil) // ShellHandler: runs the real command locally
	for _, name := range []string{"uname", "kernel", "uptime"} {
		stdout, stderr, code := fleet.run(t, []string{name}, "--host", "s1")
		if exitcode.Code(code) != exitcode.OK {
			t.Errorf("%s: code = %d, stderr = %s", name, code, stderr)
		}
		if len(stdout) < len("=== s1: ok") {
			t.Errorf("%s: stdout = %q", name, stdout)
		}
	}
}

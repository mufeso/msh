package app

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/tmux"
	"example.com/msh/internal/tmux/tmuxtest"
)

// runTmux runs msh with a fake lookPath and an in-memory tmux. When
// tmuxFound is false, tmux "is not installed".
func runTmux(t *testing.T, fleet *testFleet, tmuxFound bool, args ...string) (string, string, exitcode.Code) {
	t.Helper()
	stdout, stderr, code, _ := runTmuxFake(t, fleet, tmuxFound, tmuxtest.NewFake(), args...)
	return stdout, stderr, code
}

func runTmuxFake(t *testing.T, fleet *testFleet, tmuxFound bool, fake *tmuxtest.Fake, args ...string) (string, string, exitcode.Code, *tmuxtest.Fake) {
	t.Helper()
	lookPath := func(string) (string, error) {
		if tmuxFound {
			return "/usr/bin/tmux", nil
		}
		return "", errors.New("not found")
	}
	configure := func(c *cli) {
		c.lookPath = lookPath
		c.newTmux = func(string) tmux.Runner { return fake }
	}
	stdout, stderr, code := runCLIWith(t, context.Background(), nil, configure, append(args, fleet.args()...)...)
	return stdout, stderr, code, fake
}

func TestTmuxPreconditions(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	if _, stderr, code := runTmux(t, fleet, false, "shell", "--all", "--tmux"); code != exitcode.Tmux || !contains(stderr, "tmux was not found") {
		t.Errorf("missing tmux: code = %v, stderr = %q", code, stderr)
	}
	if _, stderr, code := runTmux(t, fleet, true, "shell", "--host", "s1", "--tmux"); code != exitcode.Usage || !contains(stderr, "at least two") {
		t.Errorf("one server: code = %v, stderr = %q", code, stderr)
	}
	if _, stderr, code := runTmux(t, fleet, true, "shell", "--all"); code != exitcode.Usage || !contains(stderr, "use --tmux") {
		t.Errorf("no --tmux: code = %v, stderr = %q", code, stderr)
	}
	_, _, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux")
	if code != exitcode.OK || len(fake.Sessions()) != 1 || len(fake.Attached()) != 1 {
		t.Errorf("code = %v, calls = %v", code, fake.Calls())
	}
}

func TestPaneArgv(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 2, bastion: true})
	_, stderr, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux", "--tmux-detached", "--command", "top -d 1")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	created := fake.CallsTo("new-session")
	if len(created) != 1 {
		t.Fatalf("calls = %v", fake.Calls())
	}
	paneCmd := created[0][len(created[0])-1]
	for _, want := range []string{"shell --host s1", "--inventory " + fleet.inventory, "--known-hosts " + fleet.known,
		"--identity " + fleet.identity, "--bastion jump", "--command 'top -d 1'", "press Enter"} {
		if !contains(paneCmd, want) {
			t.Errorf("missing %q in:\n%s", want, paneCmd)
		}
	}
	if len(fake.Attached()) != 0 {
		t.Error("--tmux-detached attached anyway")
	}
}

func TestTmuxOpensOnePanePerServer(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	_, stderr, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux", "--tmux-detached")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if n := len(fake.CallsTo("new-session")) + len(fake.CallsTo("split-window")); n != 3 {
		t.Errorf("%d panes opened, want 3", n)
	}
}

func TestPaneCommandOpensItsOwnSession(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	_, _, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux", "--tmux-detached", "--command", "echo pane-ok")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	// Recover the argv of the second pane from its quoted command line by
	// letting a shell split it, then run it in-process (without argv[0]).
	split := fake.CallsTo("split-window")[0]
	line := split[len(split)-1]
	line = line[:strings.Index(line, "; status=$?")]
	out, err := exec.Command("/bin/sh", "-c", `for a in `+line+`; do printf '%s\0' "$a"; done`).Output()
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, argv[1:]...)
	if code != exitcode.OK || !contains(stdout, "pane-ok") {
		t.Fatalf("pane command failed: code = %v, stderr = %s", code, stderr)
	}
	if cmds := fleet.servers[1].Commands(); len(cmds) != 1 || cmds[0] != "echo pane-ok" {
		t.Errorf("s2 commands = %q (the second pane must talk to s2)", cmds)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("the second pane contacted s1")
	}
}

func TestInteractiveTaskWithTmux(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := fleetConfig(t, "  top:\n    argv: [top]\n    interactive: true\n")
	_, stderr, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "task", "run", "top", "--all", "--tmux", "--tmux-detached", "--config", cfg)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	created := fake.CallsTo("new-session")
	if len(created) != 1 || !contains(created[0][len(created[0])-1], "--command top") {
		t.Errorf("calls = %v", fake.Calls())
	}
}

func TestTmuxLayout(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	_, _, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux", "--tmux-detached", "--tmux-layout", "even-vertical")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	layouts := fake.CallsTo("select-layout")
	if last := layouts[len(layouts)-1]; last[len(last)-1] != "even-vertical" {
		t.Errorf("final layout call = %v", last)
	}
	_, stderr, code, fake := runTmuxFake(t, fleet, true, tmuxtest.NewFake(), "shell", "--all", "--tmux", "--tmux-layout", "diagonal")
	if code != exitcode.Usage || !contains(stderr, "unknown tmux layout") || len(fake.Calls()) > 1 {
		t.Errorf("bad layout: code = %v, stderr = %q, calls = %v", code, stderr, fake.Calls())
	}
}

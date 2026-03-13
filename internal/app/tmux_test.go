package app

import (
	"context"
	"errors"
	"testing"

	"example.com/msh/internal/exitcode"
)

// runTmux runs msh with a fake lookPath.
func runTmux(t *testing.T, fleet *testFleet, tmuxFound bool, args ...string) (string, string, exitcode.Code) {
	t.Helper()
	lookPath := func(string) (string, error) {
		if tmuxFound {
			return "/usr/bin/tmux", nil
		}
		return "", errors.New("not found")
	}
	return runCLIWith(t, context.Background(), nil, func(c *cli) { c.lookPath = lookPath }, append(args, fleet.args()...)...)
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
	stdout, _, code := runTmux(t, fleet, true, "shell", "--all", "--tmux")
	if code != exitcode.OK || !contains(stdout, "s1") || !contains(stdout, "s2") {
		t.Errorf("plan: code = %v, stdout = %q", code, stdout)
	}
}

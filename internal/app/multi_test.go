package app

import (
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestExecOnGroup(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--group", "all", "--command", "echo hi")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if n := strings.Count(stdout, ": ok"); n != 3 {
		t.Errorf("%d ok results:\n%s", n, stdout)
	}
}

func TestSummaryLine(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	stdout, _, _ := fleet.run(t, []string{"exec"}, "--all", "--command", "true")
	if !contains(stdout, "--- 2 servers: 2 ok in ") {
		t.Errorf("stdout = %q", stdout)
	}
}

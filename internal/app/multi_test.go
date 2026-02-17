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

func TestOneUnreachableServerDoesNotStopOthers(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	fleet.servers[1].Close() // s2 is down
	stdout, _, code := fleet.run(t, []string{"exec"}, "--all", "--command", "echo up")
	if exitcode.Code(code) != exitcode.Connection {
		t.Errorf("code = %d, want connection", code)
	}
	if strings.Count(stdout, ": ok") != 2 || !contains(stdout, "=== s2: error") {
		t.Errorf("stdout = %q", stdout)
	}

	stdout, _, _ = fleet.run(t, []string{"exec"}, "--all", "--command", "echo up", "--fail-fast")
	if !contains(stdout, "=== s3: canceled") || !contains(stdout, "not started") {
		t.Errorf("fail-fast stdout = %q", stdout)
	}
}

package app

import (
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
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

func TestProblemsLineAndExitCode(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	stdout, _, code := fleet.run(t, []string{"exec"}, "--all", "--command", `test "$(hostname)" = never`)
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d, want failure", code)
	}
	if !contains(stdout, "--- problems: s1, s2, s3") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestSlowServerTimesOutAlone(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 3, handlers: map[int]sshtest.Handler{2: sshtest.Blocking("")}})
	stdout, _, code := fleet.run(t, []string{"exec"}, "--all", "--command-timeout", "300ms", "--command", "echo fast")
	if exitcode.Code(code) != exitcode.Timeout {
		t.Errorf("code = %d, want timeout", code)
	}
	for _, want := range []string{"=== s1: ok", "=== s2: timeout", "=== s3: ok", "remote command timed out"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
}

func TestPrefixOutput(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	stdout, _, code := fleet.run(t, []string{"exec"}, "--all", "--prefix", "--command", "printf 'one\ntwo\n'")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d", code)
	}
	for _, want := range []string{"s1: one\n", "s1: two\n", "s2: one\n", "s2: two\n"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in %q", want, stdout)
		}
	}
}

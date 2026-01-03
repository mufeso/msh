package app

import (
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestExecLocal(t *testing.T) {
	stdout, stderr, code := runCLI(t, "exec", "--local", "--command", "echo from-msh")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if stdout != "from-msh\n" {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestExecRequiresCommand(t *testing.T) {
	if _, _, code := runCLI(t, "exec", "--local"); code != exitcode.Usage {
		t.Errorf("code = %v, want usage", code)
	}
}

func TestExecLocalStderr(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "exec", "--local", "--command", "echo out; echo err >&2")
	if stdout != "out\n" || stderr != "err\n" {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
}

func TestExecLocalFailure(t *testing.T) {
	_, stderr, code := runCLI(t, "exec", "--local", "--command", "exit 3")
	if code != exitcode.Failure {
		t.Errorf("code = %v, want failure", code)
	}
	if !contains(stderr, "exited with code 3") {
		t.Errorf("stderr = %q", stderr)
	}
}

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

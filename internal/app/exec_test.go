package app

import (
	"encoding/json"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

func TestExecLocal(t *testing.T) {
	stdout, stderr, code := runCLI(t, "exec", "--local", "--command", "echo from-msh")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "=== local: ok (exit 0") || !contains(stdout, "from-msh\n") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestExecRequiresCommand(t *testing.T) {
	if _, _, code := runCLI(t, "exec", "--local"); code != exitcode.Usage {
		t.Errorf("code = %v, want usage", code)
	}
}

func TestExecLocalJSON(t *testing.T) {
	stdout, _, code := runCLI(t, "exec", "--local", "--output", "json", "--command", "echo out; echo err >&2; exit 4")
	if code != exitcode.Failure {
		t.Errorf("code = %v, want failure", code)
	}
	var views []output.ResultView
	if err := json.Unmarshal([]byte(stdout), &views); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	v := views[0]
	if v.Status != "failed" || v.ExitCode != 4 || v.Stdout != "out\n" || v.Stderr != "err\n" {
		t.Errorf("view = %+v", v)
	}
}

func TestExecLocalTimeout(t *testing.T) {
	_, stderr, code := runCLI(t, "exec", "--local", "--command-timeout", "100ms", "--command", "sleep 5")
	if code != exitcode.Timeout {
		t.Errorf("code = %v, want timeout; stderr = %s", code, stderr)
	}
}

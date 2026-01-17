package app

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestExecRemote(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "echo remote-$((40+2))")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "=== s1: ok") || !contains(stdout, "remote-42") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestExecRemoteHostKeyFailureIsConnectionError(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	empty := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, _, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "true", "--known-hosts", empty)
	if exitcode.Code(code) != exitcode.Connection {
		t.Errorf("code = %d, want connection", code)
	}
	if !contains(stdout, "host key verification failed") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestExecInsecurePrintsWarning(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	empty := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "true", "--known-hosts", empty, "--insecure-ignore-host-key")
	if exitcode.Code(code) != exitcode.OK {
		t.Errorf("code = %d", code)
	}
	if !contains(stderr, "host key verification is DISABLED") {
		t.Errorf("no warning on stderr: %q", stderr)
	}
}

func TestExecRequiresSelection(t *testing.T) {
	if _, _, code := runCLI(t, "exec", "--command", "true"); code != exitcode.Usage {
		t.Errorf("code = %v", code)
	}
}

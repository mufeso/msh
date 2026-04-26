package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

// writeConfig writes a configuration file for a test.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInvalidRunbookIsConfigError(t *testing.T) {
	cfg := writeConfig(t, "runbooks:\n  broken:\n    steps:\n      - name: x\n        task: nope\n")
	_, stderr, code := runCLI(t, "config", "show", "--config", cfg)
	if code != exitcode.Config || !contains(stderr, `unknown task "nope"`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

const basicRunbooks = `tasks:
  hello:
    argv: [echo, hello]
runbooks:
  greet:
    description: Say hello twice
    steps:
      - name: first
        task: hello
      - name: second
        command: "echo second | cat"
        read_only: true
`

func TestRunbookList(t *testing.T) {
	stdout, stderr, code := runCLI(t, "runbook", "list", "--config", writeConfig(t, basicRunbooks))
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "greet") || !contains(stdout, "Say hello twice") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunbookShow(t *testing.T) {
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := runCLI(t, "runbook", "show", "greet", "--config", cfg)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"1. first", "task: hello", "2. second", "command: echo second | cat"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	_, stderr, code = runCLI(t, "runbook", "show", "gret", "--config", cfg)
	if code != exitcode.Usage || !contains(stderr, `Did you mean "greet"?`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestRunbookDryRunContactsNobody(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all", "--dry-run"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"2 steps on 2 servers", "Servers: s1, s2", "$ echo hello", "$ sh -c 'echo second | cat'"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	for _, s := range fleet.servers {
		if len(s.Commands()) != 0 {
			t.Error("--dry-run contacted a server")
		}
	}
}

func TestRunbookRun(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"=== s1: ok", "--- step first: ok", "hello\n", "--- step second: ok", "second\n"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if len(fleet.servers[1].Commands()) != 2 {
		t.Errorf("s2 commands = %q", fleet.servers[1].Commands())
	}
}

func TestRunbookStopsOnFailure(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := writeConfig(t, `runbooks:
  bad:
    steps:
      - name: fail
        command: "exit 3"
        read_only: true
      - name: never
        command: "echo never"
        read_only: true
`)
	stdout, _, code := fleet.run(t, []string{"runbook", "run", "bad", "--host", "s1"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d", code)
	}
	if contains(stdout, "never") || !contains(stdout, "--- step fail: failed (exit 3)") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunbookJSONSteps(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all", "--workers", "3", "--output", "json"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var run output.RunView
	if err := json.Unmarshal([]byte(stdout), &run); err != nil {
		t.Fatal(err)
	}
	if len(run.Results) != 3 {
		t.Fatalf("results = %d", len(run.Results))
	}
	for _, r := range run.Results {
		if len(r.Steps) != 2 || r.Steps[0].Name != "first" || r.Steps[1].Stdout != "second\n" {
			t.Errorf("steps = %+v", r.Steps)
		}
	}
}

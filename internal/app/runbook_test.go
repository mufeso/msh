package app

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
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

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

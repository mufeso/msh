package app

import (
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestConfigProvidesInventoryAndUser(t *testing.T) {
	// web02 has no user in the inventory, but the inventory defaults set
	// "admin", which wins over the configuration's fallback user.
	servers := func(args ...string) string {
		stdout, stderr, code := runCLI(t, append([]string{"hosts", "list"}, args...)...)
		if code != exitcode.OK {
			t.Fatalf("code = %v, stderr = %s", code, stderr)
		}
		return stdout
	}
	out := servers("--config", "testdata/config.json", "--host", "db01")
	if want := "dbadmin"; !contains(out, want) {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestMissingConfigFileIsIOError(t *testing.T) {
	_, _, code := runCLI(t, "hosts", "list", "--config", "testdata/nope.json")
	if code != exitcode.IO {
		t.Errorf("code = %v, want %v", code, exitcode.IO)
	}
}

// contains is shorthand for strings.Contains in assertions.
func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestInvalidConfigIsConfigError(t *testing.T) {
	_, stderr, code := runCLI(t, "hosts", "list", "--inventory", "testdata/inventory.yaml", "--config", "testdata/bad-config.yaml")
	if code != exitcode.Config {
		t.Errorf("code = %v, want %v", code, exitcode.Config)
	}
	if !contains(stderr, "port: 70000 is out of range") {
		t.Errorf("stderr = %q", stderr)
	}
}

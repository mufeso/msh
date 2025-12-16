package app

import (
	"os"
	"path/filepath"
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

func TestFlagOverridesConfigFile(t *testing.T) {
	// testdata/config.json sets user "fromconfig"; web01 has no user of its
	// own but the inventory default "admin" applies to it, so use a
	// generated inventory without defaults.
	dir := t.TempDir()
	inv := filepath.Join(dir, "inv.yaml")
	if err := os.WriteFile(inv, []byte("servers:\n  - name: a\n    address: 192.0.2.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(cfg, []byte("user: fromconfig\nport: 2222\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runCLI(t, "hosts", "list", "--config", cfg, "--inventory", inv, "--user", "fromflag")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "fromflag") {
		t.Errorf("flag did not win: %q", stdout)
	}

	stdout, _, _ = runCLI(t, "hosts", "list", "--config", cfg, "--inventory", inv)
	if !contains(stdout, "fromconfig") {
		t.Errorf("config file value missing: %q", stdout)
	}
}

func TestEnvironmentPrecedence(t *testing.T) {
	dir := t.TempDir()
	inv := filepath.Join(dir, "inv.yaml")
	if err := os.WriteFile(inv, []byte("servers:\n  - name: a\n    address: 192.0.2.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(cfg, []byte("user: fromconfig\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"MSH_CONFIG": cfg, "MSH_INVENTORY": inv, "MSH_USER": "fromenv"}

	stdout, stderr, code := runCLIEnv(t, env, "hosts", "list")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "fromenv") {
		t.Errorf("environment did not beat config file: %q", stdout)
	}
	stdout, _, _ = runCLIEnv(t, env, "hosts", "list", "--user", "fromflag")
	if !contains(stdout, "fromflag") {
		t.Errorf("flag did not beat environment: %q", stdout)
	}
}

func TestDefaultConfigFileIsOptionalButUsed(t *testing.T) {
	home := t.TempDir()
	inv := filepath.Join(home, "inv.yaml")
	if err := os.WriteFile(inv, []byte("servers:\n  - name: a\n    address: 192.0.2.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// No config file yet: still works.
	if _, stderr, code := runCLIEnv(t, map[string]string{"HOME": home}, "hosts", "list", "--inventory", inv); code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	// Create ~/.config/msh/config.yaml and check it is picked up.
	cfgDir := filepath.Join(home, ".config", "msh")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := "inventory: " + inv + "\nuser: fromdefaultfile\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLIEnv(t, map[string]string{"HOME": home}, "hosts", "list")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "fromdefaultfile") {
		t.Errorf("default config not used: %q", stdout)
	}
}

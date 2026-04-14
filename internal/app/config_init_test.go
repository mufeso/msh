package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestConfigInit(t *testing.T) {
	home := t.TempDir()
	env := map[string]string{"HOME": home}
	if _, stderr, code := runCLIEnv(t, env, "config", "init"); code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	dir := filepath.Join(home, ".config", "msh")
	for _, name := range []string{"config.yaml", "inventory.yaml"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v, mode %v", name, err, info)
		}
	}
	// The written files are valid as they are.
	if _, stderr, code := runCLIEnv(t, env, "config", "validate"); code != exitcode.OK {
		t.Errorf("validate after init: code = %v, stderr = %s", code, stderr)
	}
	// A second init must not overwrite.
	os.WriteFile(filepath.Join(dir, "inventory.yaml"), []byte("# mine\n"), 0o600)
	if _, stderr, code := runCLIEnv(t, env, "config", "init"); code != exitcode.Usage || !contains(stderr, "already exists") {
		t.Errorf("overwrite: code = %v, stderr = %q", code, stderr)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "inventory.yaml")); string(data) != "# mine\n" {
		t.Error("existing file was overwritten without --force")
	}
	if _, _, code := runCLIEnv(t, env, "config", "init", "--force"); code != exitcode.OK {
		t.Errorf("--force: code = %v", code)
	}
}

func TestTemplatesMatchExamples(t *testing.T) {
	for _, name := range []string{"config.yaml", "inventory.yaml"} {
		embedded, err := templates.ReadFile("templates/" + name)
		if err != nil {
			t.Fatal(err)
		}
		example, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(embedded, example) {
			t.Errorf("internal/app/templates/%s differs from examples/%s; copy it over", name, name)
		}
	}
}

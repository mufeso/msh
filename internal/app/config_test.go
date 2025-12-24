package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestConfigShowSources(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(cfg, []byte("workers: 8\nmode: sequential\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"MSH_CONFIG": cfg, "MSH_MODE": "parallel"}
	stdout, stderr, code := runCLIEnv(t, env, "config", "show", "--output", "json", "--port", "2200")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var views []settingView
	if err := json.Unmarshal([]byte(stdout), &views); err != nil {
		t.Fatal(err)
	}
	got := map[string]settingView{}
	for _, v := range views {
		got[v.Key] = v
	}
	want := map[string]settingView{
		"workers": {"workers", "8", "config file"},
		"mode":    {"mode", "parallel", "environment"},
		"port":    {"port", "2200", "flag"},
		"output":  {"output", "json", "flag"},
		"user":    {"user", "", "default"},
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %+v, want %+v", k, got[k], w)
		}
	}
}

func TestConfigShowText(t *testing.T) {
	stdout, _, code := runCLI(t, "config", "show")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	if !contains(stdout, "connect_timeout") || !contains(stdout, "10s") {
		t.Errorf("stdout = %q", stdout)
	}
}

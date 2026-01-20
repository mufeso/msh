package app

import (
	"encoding/json"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

func bastionsOf(t *testing.T, args ...string) map[string]string {
	t.Helper()
	all := append([]string{"hosts", "list", "--inventory", "testdata/bastion.yaml", "--output", "json"}, args...)
	stdout, stderr, code := runCLI(t, all...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var servers []inventory.Server
	if err := json.Unmarshal([]byte(stdout), &servers); err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, s := range servers {
		m[s.Name] = s.Bastion
	}
	return m
}

func TestBastionPrecedenceThroughCLI(t *testing.T) {
	got := bastionsOf(t)
	want := map[string]string{"app01": "jump2", "app02": "jump1", "edge01": ""}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("default precedence: %s via %q, want %q", k, got[k], v)
		}
	}
	got = bastionsOf(t, "--bastion", "jump1")
	for _, name := range []string{"app01", "app02", "edge01"} {
		if got[name] != "jump1" {
			t.Errorf("--bastion jump1: %s via %q", name, got[name])
		}
	}
	got = bastionsOf(t, "--bastion", "none")
	for _, name := range []string{"app01", "app02", "edge01"} {
		if got[name] != "" {
			t.Errorf("--bastion none: %s via %q", name, got[name])
		}
	}
}

func TestUnknownBastionFlag(t *testing.T) {
	_, stderr, code := runCLI(t, "hosts", "list", "--inventory", "testdata/bastion.yaml", "--bastion", "ghost")
	if code != exitcode.Usage || !contains(stderr, `unknown bastion "ghost"`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

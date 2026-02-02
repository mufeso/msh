package configfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsAndTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `user: ops
workers: 4
tasks:
  uptime:
    description: Show uptime
    argv: [uptime]
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.User == nil || *f.User != "ops" || f.Workers == nil || *f.Workers != 4 {
		t.Errorf("settings = %+v", f.Layer)
	}
	if tk, ok := f.Tasks["uptime"]; !ok || tk.Description != "Show uptime" {
		t.Errorf("tasks = %+v", f.Tasks)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("tasks:\n  a:\n    comand: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("unknown task key accepted")
	}
}

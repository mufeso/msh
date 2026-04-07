package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/sshtest"
)

func TestReportFile(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 3, handlers: map[int]sshtest.Handler{1: sshtest.Static("", "", 2)}})
	report := filepath.Join(t.TempDir(), "run.json")
	_, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--command", "echo hi", "--report", report)
	if exitcode.Code(code) != exitcode.Failure {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	info, err := os.Stat(report)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(report)
	var doc output.RunView
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"s1", "s2", "s3"} {
		if doc.Results[i].Server != name {
			t.Errorf("report result %d = %s, want %s (selection order)", i, doc.Results[i].Server, name)
		}
	}
	if doc.Summary.Failed != 1 || doc.Summary.OK != 2 {
		t.Errorf("summary = %+v", doc.Summary)
	}
}

func TestReportFileUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}
	fleet := newTestFleet(t, 1, nil)
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	_, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--command", "true", "--report", filepath.Join(dir, "r.json"))
	if exitcode.Code(code) != exitcode.IO || !contains(stderr, "cannot write file") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files left behind: %v", entries)
	}
}

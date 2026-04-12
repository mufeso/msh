package app

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

// Paths are relative to this package directory (internal/app).
var (
	exampleConfig    = filepath.Join("..", "..", "examples", "config.yaml")
	exampleInventory = filepath.Join("..", "..", "examples", "inventory.yaml")
	exampleScript    = filepath.Join("..", "..", "examples", "scripts", "health-check.sh")
)

func TestExamplesValidate(t *testing.T) {
	stdout, stderr, code := runCLI(t, "config", "validate", "--config", exampleConfig, "--inventory", exampleInventory, "--output", "json")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var sum validationSummary
	if err := json.Unmarshal([]byte(stdout), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Servers != 5 {
		t.Errorf("servers = %d", sum.Servers)
	}
}

func TestExampleTasks(t *testing.T) {
	stdout, stderr, code := runCLI(t, "task", "list", "--config", exampleConfig, "--output", "json")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var tasks []taskSummary
	json.Unmarshal([]byte(stdout), &tasks)
	if len(tasks) != 5 {
		t.Errorf("tasks = %+v", tasks)
	}
	for _, tk := range tasks {
		if _, _, code := runCLI(t, "task", "show", tk.Name, "--config", exampleConfig); code != exitcode.OK {
			t.Errorf("task show %s: code = %v", tk.Name, code)
		}
	}
}

func TestExampleInventorySelections(t *testing.T) {
	stdout, _, code := runCLI(t, "hosts", "list", "--inventory", exampleInventory, "--group", "web", "--tag", "environment=production", "--output", "csv")
	if code != exitcode.OK || !contains(stdout, "web01,") || !contains(stdout, "web02,") || contains(stdout, "test01") {
		t.Errorf("code = %v, stdout = %q", code, stdout)
	}
}

func TestExampleScriptRuns(t *testing.T) {
	fleet := newTestFleet(t, 1, nil) // runs the script with the local shell
	stdout, stderr, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", exampleScript, "--interpreter", "bash")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"== host", "== disk", "== service sshd"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
}

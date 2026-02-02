package app

import (
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestInvalidTasksAreConfigErrors(t *testing.T) {
	_, stderr, code := runCLI(t, "config", "show", "--config", "testdata/bad-tasks.yaml")
	if code != exitcode.Config || !contains(stderr, `task "broken"`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestValidTasksLoad(t *testing.T) {
	if _, stderr, code := runCLI(t, "config", "show", "--config", "testdata/tasks.yaml"); code != exitcode.OK {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestTaskList(t *testing.T) {
	stdout, stderr, code := runCLI(t, "task", "list", "--config", "testdata/tasks.yaml")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"message", "msg", "shell", "restart-service", "changes", "security-summary", "script", "uptime", "direct"} {
		if !contains(stdout, want) {
			t.Errorf("task list missing %q:\n%s", want, stdout)
		}
	}
}

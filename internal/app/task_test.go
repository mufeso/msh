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

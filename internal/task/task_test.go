package task

import "testing"

func TestMode(t *testing.T) {
	tests := []struct {
		task Task
		want Mode
	}{
		{Task{Argv: []string{"uptime"}}, ModeDirect},
		{Task{Command: "df -h | sort"}, ModeShell},
		{Task{Script: "echo hi\n"}, ModeScript},
	}
	for _, tt := range tests {
		if got := tt.task.Mode(); got != tt.want {
			t.Errorf("Mode() = %s, want %s", got, tt.want)
		}
	}
}

func TestValidateShape(t *testing.T) {
	bad := []Task{
		{},
		{Command: "a", Script: "b"},
		{Argv: []string{"x"}, Interpreter: "bash"},
		{Command: "a", Safety: "yolo"},
		{Command: "a", Output: "xml"},
	}
	for i, tk := range bad {
		if err := tk.validateShape(); err == nil {
			t.Errorf("case %d: invalid task accepted: %+v", i, tk)
		}
	}
	good := Task{Command: "uptime"}
	if err := good.validateShape(); err != nil {
		t.Errorf("valid task rejected: %v", err)
	}
	if good.EffectiveSafety() != SafetyReadOnly {
		t.Error("default safety must be read-only")
	}
}

func TestScriptTasksCannotBeInteractive(t *testing.T) {
	tk := Task{Script: "echo\n", Interactive: true}
	if err := tk.validateShape(); err == nil {
		t.Error("interactive script task accepted")
	}
}

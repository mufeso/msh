package runbook

import (
	"testing"

	"example.com/msh/internal/task"
)

func TestSafety(t *testing.T) {
	tasks := testTasks(t)
	tests := []struct {
		steps []Step
		want  task.Safety
	}{
		{[]Step{{Name: "a", Task: "uptime"}}, task.SafetyReadOnly},
		{[]Step{{Name: "a", Command: "df", ReadOnly: true}}, task.SafetyReadOnly},
		{[]Step{{Name: "a", Command: "df"}}, task.SafetyChanges},
		{[]Step{{Name: "a", Task: "uptime"}, {Name: "b", Task: "restart"}}, task.SafetyChanges},
	}
	for i, tt := range tests {
		rb := Runbook{Steps: tt.steps}
		if got := rb.Safety(tasks); got != tt.want {
			t.Errorf("case %d: Safety = %s, want %s", i, got, tt.want)
		}
	}
}

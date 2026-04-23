package runbook

import "testing"

func TestPlan(t *testing.T) {
	tasks := testTasks(t)
	rb := Runbook{Steps: []Step{
		{Name: "up", Task: "uptime"},
		{Name: "restart", Task: "restart", Params: map[string]string{"service": "my app"}},
		{Name: "check", Command: "df -h | sort"},
	}}
	plan, err := rb.Plan(tasks, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"uptime", "systemctl restart 'my app'", "sh -c 'df -h | sort'"}
	for i, w := range want {
		if plan[i].Command != w {
			t.Errorf("step %d = %q, want %q", i, plan[i].Command, w)
		}
	}
}

func TestPlanMissingRequiredParam(t *testing.T) {
	rb := Runbook{Steps: []Step{{Name: "restart", Task: "restart"}}}
	if _, err := rb.Plan(testTasks(t), nil); err == nil {
		t.Error("a missing task argument was accepted")
	}
}

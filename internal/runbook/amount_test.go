package runbook

import "testing"

func TestAmount(t *testing.T) {
	tests := []struct {
		in    string
		total int
		want  int
	}{
		{"2", 10, 2},
		{"25%", 10, 3},
		{"100%", 7, 7},
		{"1%", 7, 1},
		{"0", 7, 0},
		{"0%", 7, 0},
	}
	for _, tt := range tests {
		a, err := ParseAmount(tt.in)
		if err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
		if got := a.Of(tt.total); got != tt.want {
			t.Errorf("%q of %d = %d, want %d", tt.in, tt.total, got, tt.want)
		}
	}
	for _, bad := range []string{"", "x", "-1", "101%", "1.5"} {
		if _, err := ParseAmount(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestRolloutSettingsValidated(t *testing.T) {
	steps := []Step{{Name: "a", Task: "uptime"}}
	for name, rb := range map[string]Runbook{
		"batch":    {Batch: "lots", Steps: steps},
		"zero":     {Batch: "0", Steps: steps},
		"failures": {MaxFailures: "200%", Steps: steps},
		"canary":   {Canary: -1, Steps: steps},
	} {
		if _, err := NewSet(map[string]Runbook{"r": rb}, testTasks(t)); err == nil {
			t.Errorf("%s: invalid rollout settings accepted", name)
		}
	}
	good := Runbook{Batch: "25%", MaxFailures: "1", Canary: 1, Steps: steps}
	if _, err := NewSet(map[string]Runbook{"r": good}, testTasks(t)); err != nil {
		t.Error(err)
	}
}

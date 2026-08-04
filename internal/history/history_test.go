package history

import (
	"strings"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/runner"
)

func sampleReport(servers ...string) runner.Report {
	var results []executor.Result
	for _, s := range servers {
		results = append(results, executor.Result{Server: s, Status: executor.StatusOK, Stdout: "hello from " + s + "\n"})
	}
	summary := runner.Summarize(results)
	summary.Duration = 2 * time.Second
	return runner.Report{Results: results, Summary: summary}
}

func TestNewRecord(t *testing.T) {
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	r := NewRecord([]string{"exec", "--all"}, "ops", sampleReport("a", "b"), now)
	if r.Schema != Schema || !strings.HasPrefix(r.ID, "20260304T050605.000000-") || len(r.Results) != 2 {
		t.Errorf("record = %+v", r)
	}
	if other := NewRecord([]string{"x"}, "ops", sampleReport("a"), now); other.ID == r.ID {
		t.Error("two records in the same second share an ID")
	}
}

func TestOutputIsLimited(t *testing.T) {
	report := runner.Report{Results: []executor.Result{{
		Server: "a", Stdout: strings.Repeat("x", MaxOutput*2),
		Steps: []executor.StepResult{{Name: "s", Stdout: strings.Repeat("y", MaxOutput*2)}},
	}}}
	r := NewRecord([]string{"c"}, "ops", report, time.Now())
	if len(r.Results[0].Stdout) > MaxOutput+100 || !strings.HasSuffix(r.Results[0].Stdout, "[truncated in history]\n") {
		t.Errorf("stdout length = %d", len(r.Results[0].Stdout))
	}
	if len(r.Results[0].Steps[0].Stdout) > MaxOutput+100 {
		t.Error("step output was not limited")
	}
	if len(report.Results[0].Steps[0].Stdout) != MaxOutput*2 {
		t.Error("the caller's data was changed")
	}
}

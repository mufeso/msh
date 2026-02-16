package runner

import (
	"slices"
	"testing"
	"time"

	"example.com/msh/internal/executor"
)

func TestSummarize(t *testing.T) {
	r := func(name string, st executor.Status) executor.Result {
		return executor.Result{Server: name, Status: st}
	}
	s := Summarize([]executor.Result{
		r("a", executor.StatusOK), r("b", executor.StatusFailed), r("c", executor.StatusError),
		r("d", executor.StatusTimeout), r("e", executor.StatusOK), r("f", executor.StatusCanceled),
	})
	if s.Total != 6 || s.OK != 2 || s.Failed != 1 || s.Error != 1 || s.Timeout != 1 || s.Canceled != 1 {
		t.Errorf("summary = %+v", s)
	}
	if !slices.Equal(s.Problems, []string{"b", "c", "d", "f"}) || s.Succeeded() {
		t.Errorf("problems = %v", s.Problems)
	}
}

func TestSummaryString(t *testing.T) {
	s := Summary{Total: 3, OK: 2, Failed: 1, Duration: 1234 * time.Millisecond}
	if got := s.String(); got != "3 servers: 2 ok, 1 failed in 1.23s" {
		t.Errorf("String() = %q", got)
	}
	s = Summary{Total: 1, OK: 1, Duration: 5 * time.Millisecond}
	if got := s.String(); got != "1 server: 1 ok in 10ms" {
		t.Errorf("String() = %q", got)
	}
}

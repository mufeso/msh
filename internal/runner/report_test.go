package runner

import (
	"slices"
	"testing"

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

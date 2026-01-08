package app

import (
	"testing"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
)

func TestExitCodeForResults(t *testing.T) {
	r := func(st executor.Status) executor.Result { return executor.Result{Status: st} }
	tests := []struct {
		name    string
		results []executor.Result
		want    exitcode.Code
	}{
		{"none", nil, exitcode.OK},
		{"all ok", []executor.Result{r(executor.StatusOK), r(executor.StatusOK)}, exitcode.OK},
		{"one failed", []executor.Result{r(executor.StatusOK), r(executor.StatusFailed)}, exitcode.Failure},
		{"timeout beats failure", []executor.Result{r(executor.StatusFailed), r(executor.StatusTimeout)}, exitcode.Timeout},
		{"canceled beats all", []executor.Result{r(executor.StatusTimeout), r(executor.StatusCanceled)}, exitcode.Canceled},
	}
	for _, tt := range tests {
		if got := exitCodeForResults(tt.results); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

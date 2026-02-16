package runner

import (
	"fmt"
	"strings"
	"time"

	"example.com/msh/internal/executor"
)

// Summary counts results by status.
type Summary struct {
	Total    int      `json:"total"`
	OK       int      `json:"ok"`
	Failed   int      `json:"failed"`
	Error    int      `json:"error"`
	Timeout  int      `json:"timeout"`
	Canceled int      `json:"canceled"`
	Problems []string `json:"problems"` // servers that did not succeed, in selection order

	// Duration is the wall-clock time of the whole run. In parallel mode
	// it is less than the sum of the per-server durations.
	Duration time.Duration `json:"-"`
}

// String returns a one-line description such as
// "3 servers: 2 ok, 1 failed in 1.24s".
func (s Summary) String() string {
	parts := []string{fmt.Sprintf("%d ok", s.OK)}
	for _, p := range []struct {
		n     int
		label string
	}{{s.Failed, "failed"}, {s.Error, "error"}, {s.Timeout, "timeout"}, {s.Canceled, "canceled"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.label))
		}
	}
	noun := "servers"
	if s.Total == 1 {
		noun = "server"
	}
	return fmt.Sprintf("%d %s: %s in %s", s.Total, noun, strings.Join(parts, ", "), s.Duration.Round(10*time.Millisecond))
}

// Succeeded reports whether every server succeeded.
func (s Summary) Succeeded() bool { return s.Total == s.OK }

// Report is the outcome of a run.
type Report struct {
	Results []executor.Result // in selection order
	Summary Summary
}

// Summarize counts results.
func Summarize(results []executor.Result) Summary {
	s := Summary{Total: len(results), Problems: []string{}}
	for _, r := range results {
		switch r.Status {
		case executor.StatusOK:
			s.OK++
		case executor.StatusFailed:
			s.Failed++
		case executor.StatusTimeout:
			s.Timeout++
		case executor.StatusCanceled:
			s.Canceled++
		default:
			s.Error++
		}
		if r.Status != executor.StatusOK {
			s.Problems = append(s.Problems, r.Server)
		}
	}
	return s
}

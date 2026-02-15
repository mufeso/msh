package runner

import "example.com/msh/internal/executor"

// Summary counts results by status.
type Summary struct {
	Total    int      `json:"total"`
	OK       int      `json:"ok"`
	Failed   int      `json:"failed"`
	Error    int      `json:"error"`
	Timeout  int      `json:"timeout"`
	Canceled int      `json:"canceled"`
	Problems []string `json:"problems"` // servers that did not succeed, in selection order
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

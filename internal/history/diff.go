package history

import (
	"strconv"
	"strings"

	"example.com/msh/internal/output"
)

// Change is one difference between two runs.
type Change struct {
	Server string `json:"server"`
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// Diff compares two runs server by server, in the order of a, then the
// servers only b has. Timing is ignored: it differs in every run.
func Diff(a, b Record) []Change {
	after := map[string]output.ResultView{}
	for _, r := range b.Results {
		after[r.Server] = r
	}
	var out []Change
	seen := map[string]bool{}
	for _, x := range a.Results {
		seen[x.Server] = true
		y, ok := after[x.Server]
		if !ok {
			out = append(out, Change{x.Server, "presence", "in run", "not in run"})
			continue
		}
		if x.Status != y.Status {
			out = append(out, Change{x.Server, "status", x.Status, y.Status})
		}
		if x.ExitCode != y.ExitCode {
			out = append(out, Change{x.Server, "exit_code", strconv.Itoa(x.ExitCode), strconv.Itoa(y.ExitCode)})
		}
		if x.Stdout != y.Stdout {
			before, afterLine := firstDifference(x.Stdout, y.Stdout)
			out = append(out, Change{x.Server, "stdout", before, afterLine})
		}
	}
	for _, y := range b.Results {
		if !seen[y.Server] {
			out = append(out, Change{y.Server, "presence", "not in run", "in run"})
		}
	}
	return out
}

// firstDifference returns the first lines that differ.
func firstDifference(a, b string) (string, string) {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < max(len(x), len(y)); i++ {
		l, r := lineAt(x, i), lineAt(y, i)
		if l != r {
			return l, r
		}
	}
	return "", ""
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

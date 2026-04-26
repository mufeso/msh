package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"example.com/msh/internal/executor"
)

// ResultView is the stable machine-readable form of an executor.Result.
// The JSON field names are part of msh's public contract.
type ResultView struct {
	Server          string    `json:"server"`
	Address         string    `json:"address"`
	Status          string    `json:"status"`
	ExitCode        int       `json:"exit_code"`
	Stdout          string    `json:"stdout"`
	Stderr          string    `json:"stderr"`
	StdoutTruncated bool      `json:"stdout_truncated"`
	StderrTruncated bool      `json:"stderr_truncated"`
	Error           string    `json:"error,omitempty"`
	Started         time.Time `json:"started"`
	DurationMS      int64     `json:"duration_ms"`

	// Steps is set for runbooks only.
	Steps []executor.StepResult `json:"steps,omitempty"`
}

// NewResultView converts a result into its view.
func NewResultView(r executor.Result) ResultView {
	v := ResultView{
		Server:          r.Server,
		Address:         r.Address,
		Status:          string(r.Status),
		ExitCode:        r.ExitCode,
		Stdout:          r.Stdout,
		Stderr:          r.Stderr,
		StdoutTruncated: r.StdoutTruncated,
		StderrTruncated: r.StderrTruncated,
		Started:         r.Started.UTC(),
		DurationMS:      r.Duration.Milliseconds(),
		Steps:           r.Steps,
	}
	if r.Err != nil {
		v.Error = r.Err.Error()
	}
	return v
}

// WriteResults writes results in the given format.
func WriteResults(w io.Writer, format Format, results []executor.Result) error {
	if format == JSON {
		views := make([]ResultView, 0, len(results))
		for _, r := range results {
			views = append(views, NewResultView(r))
		}
		return WriteJSON(w, views)
	}
	for _, r := range results {
		if err := WriteResultText(w, r); err != nil {
			return err
		}
	}
	return nil
}

// WriteResultText writes one result as a human-readable block:
//
//	=== web01: ok (exit 0, 120ms)
//	...stdout...
//	--- stderr
//	...stderr...
func WriteResultText(w io.Writer, r executor.Result) error {
	var b strings.Builder
	fmt.Fprintf(&b, "=== %s: %s", r.Server, r.Status)
	if r.ExitCode >= 0 {
		fmt.Fprintf(&b, " (exit %d, %s)\n", r.ExitCode, r.Duration.Round(time.Millisecond))
	} else {
		fmt.Fprintf(&b, " (%s)\n", r.Duration.Round(time.Millisecond))
	}
	if r.Err != nil {
		fmt.Fprintf(&b, "error: %v\n", r.Err)
	}
	writeStream(&b, r.Stdout, r.StdoutTruncated)
	if r.Stderr != "" {
		b.WriteString("--- stderr\n")
		writeStream(&b, r.Stderr, r.StderrTruncated)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeStream writes captured output, ensuring it ends with a newline.
func writeStream(b *strings.Builder, s string, truncated bool) {
	if s == "" {
		return
	}
	b.WriteString(s)
	if !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
	if truncated {
		b.WriteString("[output truncated]\n")
	}
}

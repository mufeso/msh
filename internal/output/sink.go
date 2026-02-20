package output

import (
	"fmt"
	"io"
	"strings"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/runner"
)

// Sink receives results as servers finish and a summary at the end.
// Implementations are used from a single goroutine.
type Sink interface {
	Write(r executor.Result) error
	Close(s runner.Summary) error
}

// SinkOptions tune text output.
type SinkOptions struct {
	Prefix bool // one "server: line" per output line instead of blocks
}

// NewSink returns the sink for a format.
func NewSink(w io.Writer, format Format, opts SinkOptions) Sink {
	if format == JSON {
		return &jsonSink{w: w}
	}
	return &textSink{w: w, prefix: opts.Prefix}
}

// textSink prints human-readable output immediately.
type textSink struct {
	w      io.Writer
	prefix bool
}

func (s *textSink) Write(r executor.Result) error {
	if s.prefix {
		return writePrefixed(s.w, r)
	}
	return WriteResultText(s.w, r)
}

func (s *textSink) Close(sum runner.Summary) error {
	if sum.Total <= 1 {
		return nil
	}
	if _, err := fmt.Fprintf(s.w, "--- %s\n", sum); err != nil {
		return err
	}
	if len(sum.Problems) > 0 {
		_, err := fmt.Fprintf(s.w, "--- problems: %s\n", strings.Join(sum.Problems, ", "))
		return err
	}
	return nil
}

// writePrefixed writes every output line prefixed with the server name,
// and a status line for servers that did not succeed.
func writePrefixed(w io.Writer, r executor.Result) error {
	var b strings.Builder
	for _, line := range lines(r.Stdout) {
		fmt.Fprintf(&b, "%s: %s\n", r.Server, line)
	}
	for _, line := range lines(r.Stderr) {
		fmt.Fprintf(&b, "%s [stderr]: %s\n", r.Server, line)
	}
	switch {
	case r.Status == executor.StatusFailed:
		fmt.Fprintf(&b, "%s: [failed: exit %d]\n", r.Server, r.ExitCode)
	case r.Status != executor.StatusOK:
		fmt.Fprintf(&b, "%s: [%s: %v]\n", r.Server, r.Status, r.Err)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// lines splits output into lines; a final newline does not produce an
// empty extra line.
func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// jsonSink collects results and writes one JSON document at the end.
type jsonSink struct {
	w     io.Writer
	views []ResultView
}

func (s *jsonSink) Write(r executor.Result) error {
	s.views = append(s.views, NewResultView(r))
	return nil
}

func (s *jsonSink) Close(runner.Summary) error {
	if s.views == nil {
		s.views = []ResultView{}
	}
	return WriteJSON(s.w, s.views)
}

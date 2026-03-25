package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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
	switch format {
	case JSON:
		return &jsonSink{w: w}
	case CSV:
		return &csvSink{w: csv.NewWriter(w)}
	case JSONL:
		return &jsonlSink{w: w}
	default:
		return &textSink{w: w, prefix: opts.Prefix}
	}
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

func (s *jsonSink) Close(sum runner.Summary) error {
	if s.views == nil {
		s.views = []ResultView{}
	}
	return WriteJSON(s.w, RunView{Results: s.views, Summary: NewSummaryView(sum)})
}

// RunView is the JSON document of a run.
type RunView struct {
	Results []ResultView `json:"results"`
	Summary SummaryView  `json:"summary"`
}

// SummaryView is the JSON form of runner.Summary. Embedding runner.Summary
// reuses its JSON tags; DurationMS adds the duration as a number.
type SummaryView struct {
	runner.Summary
	DurationMS int64 `json:"duration_ms"`
}

// NewSummaryView converts a summary.
func NewSummaryView(s runner.Summary) SummaryView {
	if s.Problems == nil {
		s.Problems = []string{}
	}
	return SummaryView{Summary: s, DurationMS: s.Duration.Milliseconds()}
}

// csvHeader names the columns of CSV run output.
var csvHeader = []string{"server", "address", "status", "exit_code", "duration_ms", "started", "error", "stdout", "stderr"}

// csvSink writes one CSV row per result as soon as it arrives.
type csvSink struct {
	w      *csv.Writer
	header bool
}

func (s *csvSink) writeHeader() error {
	if s.header {
		return nil
	}
	s.header = true
	return s.w.Write(csvHeader)
}

func (s *csvSink) Write(r executor.Result) error {
	if err := s.writeHeader(); err != nil {
		return err
	}
	v := NewResultView(r)
	row := []string{
		v.Server, v.Address, v.Status, strconv.Itoa(v.ExitCode), strconv.FormatInt(v.DurationMS, 10),
		v.Started.Format(time.RFC3339), v.Error, v.Stdout, v.Stderr,
	}
	if err := s.w.Write(row); err != nil {
		return err
	}
	// Flush every row so the file grows while servers finish.
	s.w.Flush()
	return s.w.Error()
}

func (s *csvSink) Close(runner.Summary) error {
	if err := s.writeHeader(); err != nil {
		return err
	}
	s.w.Flush()
	return s.w.Error()
}

// jsonlSink writes one line per result immediately and a summary line at
// the end. The "type" field tells the two kinds apart.
type jsonlSink struct {
	w io.Writer
}

type jsonlResult struct {
	Type string `json:"type"`
	ResultView
}

type jsonlSummary struct {
	Type string `json:"type"`
	SummaryView
}

func (s *jsonlSink) Write(r executor.Result) error {
	return WriteJSONLine(s.w, jsonlResult{Type: "result", ResultView: NewResultView(r)})
}

func (s *jsonlSink) Close(sum runner.Summary) error {
	return WriteJSONLine(s.w, jsonlSummary{Type: "summary", SummaryView: NewSummaryView(sum)})
}

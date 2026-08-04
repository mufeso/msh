// Package history stores a record of each msh run on the local machine,
// so runs can be listed, inspected and compared later.
package history

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"example.com/msh/internal/output"
	"example.com/msh/internal/runner"
	"example.com/msh/internal/shellquote"
)

// Schema is the current version of the file format.
//
//	1  first version
//	2  adds args and operator
const Schema = 2

// MaxOutput limits the stored stdout and stderr of each server and step.
const MaxOutput = 64 << 10

// Record is one run.
type Record struct {
	Schema   int                 `json:"schema"`
	ID       string              `json:"id"`
	Started  time.Time           `json:"started"`
	Command  string              `json:"command"`
	Args     []string            `json:"args"`
	Operator string              `json:"operator"`
	Results  []output.ResultView `json:"results"`
	Summary  output.SummaryView  `json:"summary"`
}

// NewRecord builds the record of a finished run. now is the end of the
// run, and the start is derived from the run's duration.
func NewRecord(args []string, operator string, report runner.Report, now time.Time) Record {
	started := now.Add(-report.Summary.Duration).UTC()
	r := Record{
		Schema:   Schema,
		ID:       newID(started),
		Started:  started,
		Command:  "msh " + shellquote.Join(args),
		Args:     args,
		Operator: operator,
		Results:  make([]output.ResultView, 0, len(report.Results)),
		Summary:  output.NewSummaryView(report.Summary),
	}
	for _, res := range report.Results {
		v := output.NewResultView(res)
		v.Stdout, v.Stderr = trim(v.Stdout), trim(v.Stderr)
		v.Steps = append(v.Steps[:0:0], v.Steps...) // never change the caller's slice
		for i := range v.Steps {
			v.Steps[i].Stdout, v.Steps[i].Stderr = trim(v.Steps[i].Stdout), trim(v.Steps[i].Stderr)
		}
		r.Results = append(r.Results, v)
	}
	return r
}

// migrate upgrades a record read from a file, one version at a time, and
// refuses files from a newer msh, whose meaning it can't know.
func migrate(r *Record) error {
	if r.Schema > Schema {
		return fmt.Errorf("written by a newer msh (schema %d, this version reads up to %d)", r.Schema, Schema)
	}
	if r.Schema < 2 {
		// Version 1 kept only the joined command line. Splitting it again
		// is approximate, which is fine for display.
		r.Args = strings.Fields(strings.TrimPrefix(r.Command, "msh "))
		r.Operator = "unknown"
		r.Schema = 2
	}
	return nil
}

// trim limits stored output.
func trim(s string) string {
	if len(s) <= MaxOutput {
		return s
	}
	return s[:MaxOutput] + "\n[truncated in history]\n"
}

// newID returns an ID that sorts by time, down to the microsecond, so
// "last" is right even for quick runs in a script. The random part keeps
// runs apart if they start at the same microsecond.
func newID(t time.Time) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return t.Format("20060102T150405.000000") + "-" + hex.EncodeToString(b)
}

// Package output renders msh results for humans (text) and machines.
package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// newTabWriter returns a tabwriter that aligns tab-separated cells into
// columns separated by two spaces. Data is buffered until Flush.
func newTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// Table writes headers and rows as aligned columns.
func Table(w io.Writer, headers []string, rows [][]string) error {
	tw := newTabWriter(w)
	if len(headers) > 0 {
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	// Flush writes the buffered, aligned text and reports write errors.
	return tw.Flush()
}

// Pair is one line of a key/value list.
type Pair struct {
	Key   string
	Value string
}

// KeyValues writes "Key:  value" lines with aligned values.
func KeyValues(w io.Writer, pairs []Pair) error {
	tw := newTabWriter(w)
	for _, p := range pairs {
		fmt.Fprintf(tw, "%s:\t%s\n", p.Key, p.Value)
	}
	return tw.Flush()
}

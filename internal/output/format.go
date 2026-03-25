package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format is an output format. Using a defined type instead of a plain
// string documents intent and lets the compiler catch mistakes.
type Format string

// Supported formats.
const (
	Text  Format = "text"
	JSON  Format = "json"
	CSV   Format = "csv"
	JSONL Format = "jsonl" // one JSON object per line, streamed
)

// ParseFormat converts user input into a Format.
func ParseFormat(s string) (Format, error) {
	switch f := Format(s); f {
	case Text, JSON, CSV, JSONL:
		return f, nil
	default:
		return "", fmt.Errorf("invalid output format %q: must be text, json, csv or jsonl", s)
	}
}

// WriteJSON writes v as indented JSON followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	// JSON output is for programs, not web pages: keep <, > and & as they
	// are instead of escaping them as \u003c etc.
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// WriteJSONLine writes v as one compact JSON line.
func WriteJSONLine(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v) // Encode appends exactly one newline
}

// WriteObject writes a single value as indented JSON or as one JSONL
// line.
func WriteObject(w io.Writer, f Format, v any) error {
	if f == JSONL {
		return WriteJSONLine(w, v)
	}
	return WriteJSON(w, v)
}

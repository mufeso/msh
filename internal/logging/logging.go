// Package logging builds msh's diagnostic logger. Diagnostics always go to
// stderr, never to stdout, so they cannot corrupt machine-readable output.
// Logs never contain command output, script contents or secrets.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// Log formats.
const (
	FormatText = "text"
	FormatJSON = "json"
)

// ValidFormat reports whether f is a known log format.
func ValidFormat(f string) bool {
	return f == FormatText || f == FormatJSON
}

// New returns a logger writing to w. With verbose, debug messages are
// included; otherwise only warnings and errors.
func New(w io.Writer, format string, verbose bool) *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	if format == FormatJSON {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
	}
	return slog.New(&humanHandler{w: w, level: level, mu: &sync.Mutex{}})
}

// humanHandler writes "msh: message key=value ..." lines.
type humanHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	mu    *sync.Mutex // shared by all copies made by WithAttrs
}

// Enabled implements slog.Handler.
func (h *humanHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

// Handle implements slog.Handler.
func (h *humanHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString("msh: ")
	if r.Level >= slog.LevelWarn {
		b.WriteString(strings.ToLower(r.Level.String()))
		b.WriteString(": ")
	}
	b.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value.Any())
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

// WithAttrs implements slog.Handler.
func (h *humanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &c
}

// WithGroup implements slog.Handler. The human format is flat, so groups
// are ignored.
func (h *humanHandler) WithGroup(string) slog.Handler { return h }

package output

import (
	"bytes"
	"io"
	"sync"
)

// LineWriter prefixes every complete line written to it and forwards it
// to w. Several LineWriters (one per server and stream) share one mutex,
// so lines from different servers never interleave mid-line.
type LineWriter struct {
	w      io.Writer
	prefix string
	mu     *sync.Mutex
	buf    []byte // incomplete last line
}

// NewLineWriter returns a LineWriter.
func NewLineWriter(w io.Writer, prefix string, mu *sync.Mutex) *LineWriter {
	return &LineWriter{w: w, prefix: prefix, mu: mu}
}

// Write implements io.Writer. Only complete lines are written; the rest
// waits for more data or for Flush.
func (l *LineWriter) Write(p []byte) (int, error) {
	l.buf = append(l.buf, p...)
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		if err := l.emit(l.buf[:i+1]); err != nil {
			return len(p), err
		}
		l.buf = l.buf[i+1:]
	}
}

// Flush writes a final line that has no newline yet.
func (l *LineWriter) Flush() error {
	if len(l.buf) == 0 {
		return nil
	}
	line := append(l.buf, '\n')
	l.buf = nil
	return l.emit(line)
}

func (l *LineWriter) emit(line []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := io.WriteString(l.w, l.prefix); err != nil {
		return err
	}
	_, err := l.w.Write(line)
	return err
}

// LockedWriter serializes writes to w with a mutex shared with
// LineWriters, so whole-result output and live lines never interleave.
type LockedWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

// NewLockedWriter returns a LockedWriter.
func NewLockedWriter(w io.Writer, mu *sync.Mutex) *LockedWriter {
	return &LockedWriter{w: w, mu: mu}
}

// Write implements io.Writer.
func (l *LockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

package executor

import "bytes"

// DefaultOutputLimit is the default number of bytes kept per output stream.
const DefaultOutputLimit = 1 << 20 // 1 MiB

// LimitedBuffer is an io.Writer that keeps the first Limit bytes written
// to it and silently discards the rest, remembering that it did so.
//
// It always reports the full length as written. Returning a short count
// would make the writer (for example os/exec's copy goroutine) fail with
// io.ErrShortWrite and could stop the command early.
type LimitedBuffer struct {
	Limit     int
	buf       bytes.Buffer
	truncated bool
}

// NewLimitedBuffer returns a buffer that keeps at most limit bytes; a
// limit <= 0 means DefaultOutputLimit.
func NewLimitedBuffer(limit int) *LimitedBuffer {
	if limit <= 0 {
		limit = DefaultOutputLimit
	}
	return &LimitedBuffer{Limit: limit}
}

// Write implements io.Writer.
func (b *LimitedBuffer) Write(p []byte) (int, error) {
	room := b.Limit - b.buf.Len()
	if room <= 0 {
		b.truncated = len(p) > 0 || b.truncated
		return len(p), nil
	}
	if len(p) > room {
		b.buf.Write(p[:room])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

// String returns the kept bytes.
func (b *LimitedBuffer) String() string { return b.buf.String() }

// Truncated reports whether output was discarded.
func (b *LimitedBuffer) Truncated() bool { return b.truncated }

package executor

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestLimitedBuffer(t *testing.T) {
	b := NewLimitedBuffer(5)
	n, err := io.WriteString(b, "abc")
	if n != 3 || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	n, _ = io.WriteString(b, "defgh")
	if n != 5 {
		t.Errorf("Write must report the full length, got %d", n)
	}
	if b.String() != "abcde" || !b.Truncated() {
		t.Errorf("got %q truncated=%v", b.String(), b.Truncated())
	}
}

func TestLimitedBufferDefault(t *testing.T) {
	if NewLimitedBuffer(0).Limit != DefaultOutputLimit {
		t.Error("limit 0 should mean the default")
	}
}

func TestLimitedBufferWithFprintf(t *testing.T) {
	b := NewLimitedBuffer(1024)
	fmt.Fprintf(b, "%s", strings.Repeat("x", 10))
	if b.String() != "xxxxxxxxxx" || b.Truncated() {
		t.Errorf("got %q truncated=%v", b.String(), b.Truncated())
	}
}

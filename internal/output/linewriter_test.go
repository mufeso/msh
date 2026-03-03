package output

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestLineWriter(t *testing.T) {
	var out bytes.Buffer
	var mu sync.Mutex
	lw := NewLineWriter(&out, "web01: ", &mu)
	fmt.Fprint(lw, "par")
	fmt.Fprint(lw, "tial\nsecond\nthi")
	lw.Flush()
	want := "web01: partial\nweb01: second\nweb01: thi\n"
	if out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestLineWritersDoNotInterleave(t *testing.T) {
	var out bytes.Buffer
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		lw := NewLineWriter(&out, fmt.Sprintf("s%d: ", i), &mu)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				fmt.Fprintf(lw, "line %d\n", j)
			}
		}()
	}
	wg.Wait()
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if !strings.HasPrefix(line, "s") || !strings.Contains(line, ": line ") {
			t.Fatalf("interleaved line: %q", line)
		}
	}
}

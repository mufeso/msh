package exitcode

import (
	"fmt"
	"testing"
)

func TestValuesAreStable(t *testing.T) {
	// These numbers are part of the public contract of msh. Changing one
	// breaks scripts, so the test pins them.
	tests := map[Code]int{
		OK: 0, Failure: 1, Usage: 2, Config: 3, Connection: 4, Timeout: 5,
		Transfer: 6, Tmux: 7, IO: 8, Internal: 70, Canceled: 130,
	}
	for code, want := range tests {
		if int(code) != want {
			t.Errorf("%v = %d, want %d", code, int(code), want)
		}
	}
}

func TestString(t *testing.T) {
	if got := fmt.Sprint(Canceled); got != "canceled" {
		t.Errorf("fmt.Sprint(Canceled) = %q, want %q", got, "canceled")
	}
	if got := Code(42).String(); got != "unknown" {
		t.Errorf("Code(42).String() = %q, want unknown", got)
	}
}

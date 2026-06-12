package expr

import (
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		`a == "b"`, `!(a || b) && c >= 2`, `name ~ "^x"`, `"unterminated`, `((((`, `!!!!a`, `a~`, "\x00", "é == 'ü'",
	} {
		f.Add(seed)
	}
	lookup := func(string) (string, bool) { return "7", true }
	f.Fuzz(func(t *testing.T, src string) {
		e, err := Parse(src) // must never panic
		if err != nil {
			return
		}
		e.Eval(lookup) // valid expressions always evaluate
		if e.String() != src {
			t.Errorf("String() = %q, want %q", e.String(), src)
		}
	})
}

func TestDeepNestingIsRejected(t *testing.T) {
	for _, in := range []string{
		strings.Repeat("(", 100000) + "a" + strings.Repeat(")", 100000),
		strings.Repeat("!", 100000) + "a",
	} {
		_, err := Parse(in)
		if err == nil || !strings.Contains(err.Error(), "nested too deeply") {
			t.Errorf("err = %v", err)
		}
	}
	if _, err := Parse(strings.Repeat("(", 60) + "a" + strings.Repeat(")", 60)); err != nil {
		t.Errorf("reasonable nesting rejected: %v", err)
	}
}

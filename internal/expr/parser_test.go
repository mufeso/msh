package expr

import (
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	for _, in := range []string{
		`a`, `a == "b"`, `a == 1 && b != 2`, `a || b && c`, `!a`, `!(a || b)`,
		`(a == 1) || (b == 2)`, `name ~ "^db[0-9]+$"`, `name !~ "x"`, `cpus >= 4`,
	} {
		if _, err := Parse(in); err != nil {
			t.Errorf("Parse(%q) = %v", in, err)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		``:           "unexpected end",
		`a ==`:       "unexpected end",
		`(a`:         "missing )",
		`a b`:        `unexpected "b"`,
		`a == == b`:  `unexpected "=="`,
		`name ~ x`:   "quoted regular expression",
		`name ~ "["`: "missing closing",
		`&& a`:       `unexpected "&&"`,
	}
	for in, want := range tests {
		_, err := Parse(in)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}

func TestPrecedence(t *testing.T) {
	e, err := Parse(`a || b && c`)
	if err != nil {
		t.Fatal(err)
	}
	root, ok := e.root.(orNode)
	if !ok {
		t.Fatalf("root = %T, want orNode: && must bind tighter", e.root)
	}
	if _, ok := root.right.(andNode); !ok {
		t.Errorf("right = %T", root.right)
	}
}

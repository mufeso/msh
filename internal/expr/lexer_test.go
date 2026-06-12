package expr

import (
	"strings"
	"testing"
)

func kinds(tokens []token) string {
	var out []string
	for _, t := range tokens {
		out = append(out, t.text)
	}
	return strings.Join(out, " ")
}

func TestLex(t *testing.T) {
	tests := map[string]string{
		`os_family == "redhat"`: "os_family == redhat ",
		`a>=4&&b!=1||!c`:        "a >= 4 && b != 1 || ! c ",
		`name ~ '^db' `:         "name ~ ^db ",
		`tag.env !~ "x"`:        "tag.env !~ x ",
		`(cpus < 2.5)`:          "( cpus < 2.5 ) ",
		`"a \" quote"`:          `a " quote `,
	}
	for in, want := range tests {
		tokens, err := lex(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got := kinds(tokens); got != want {
			t.Errorf("lex(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLexErrors(t *testing.T) {
	for _, in := range []string{`"open`, `a @ b`, `$x`} {
		if _, err := lex(in); err == nil || !strings.Contains(err.Error(), "position") {
			t.Errorf("lex(%q): err = %v", in, err)
		}
	}
}

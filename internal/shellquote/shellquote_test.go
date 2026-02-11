package shellquote

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

var hostile = []string{
	"",
	"plain",
	"two words",
	"it's",
	`double "quotes"`,
	"$(id)",
	"`id`",
	"a; rm -rf /",
	"x | cat",
	"*",
	"~",
	"$HOME",
	"line1\nline2",
	"\\backslash",
	"'; echo pwned; '",
	"-n",
}

func TestQuote(t *testing.T) {
	tests := map[string]string{
		"":     "''",
		"abc":  "abc",
		"a b":  "'a b'",
		"it's": `'it'\''s'`,
		"$x":   "'$x'",
	}
	for in, want := range tests {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestRoundTripThroughRealShell proves the quoting with /bin/sh: every
// hostile word must arrive as exactly one unchanged argument.
func TestRoundTripThroughRealShell(t *testing.T) {
	script := `for a in "$@"; do printf '%s\0' "$a"; done`
	cmdline := "sh -c " + Quote(script) + " sh " + Join(hostile)
	out, err := exec.Command("/bin/sh", "-c", cmdline).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if !slices.Equal(got, hostile) {
		t.Errorf("round trip changed the words:\n got %q\nwant %q", got, hostile)
	}
}

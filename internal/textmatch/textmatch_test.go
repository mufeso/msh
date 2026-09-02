package textmatch

import "testing"

func TestMatch(t *testing.T) {
	const name = "xerox-workcentre-6015b"
	tests := []struct {
		pattern    string
		ignoreCase bool
		want       bool
	}{
		{"", false, true},
		{"*", false, true},
		{"xerox", false, true},
		{"workcentre", false, true},
		{"xerox*", false, true},
		{"*centre*", false, true},
		{"^xerox", false, true},
		{"^work", false, false},
		{"6015b$", false, true},
		{"xerox.*6015", false, true},
		{"XEROX", false, false},
		{"XEROX", true, true},
		{"canon", false, false},
	}
	for _, tt := range tests {
		m, err := New(tt.pattern, tt.ignoreCase)
		if err != nil {
			t.Fatalf("New(%q): %v", tt.pattern, err)
		}
		if got := m.Match(name); got != tt.want {
			t.Errorf("%q (ignore case: %v) = %v", tt.pattern, tt.ignoreCase, got)
		}
	}
}

func TestInvalidPattern(t *testing.T) {
	for _, bad := range []string{"[", "(", "a{2,1}"} {
		if _, err := New(bad, false); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

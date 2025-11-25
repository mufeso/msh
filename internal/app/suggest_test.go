package app

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"version", "version", 0},
		{"verison", "version", 2},
		{"hots", "hosts", 1},
		{"kitten", "sitting", 3},
		{"héllo", "hello", 1}, // é is one rune, two bytes
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuggest(t *testing.T) {
	candidates := []string{"help", "hosts", "version"}
	tests := []struct {
		input  string
		want   string
		wantOK bool
	}{
		{"verison", "version", true},
		{"host", "hosts", true},
		{"hlep", "help", true},
		{"completely-different", "", false},
	}
	for _, tt := range tests {
		got, ok := suggest(tt.input, candidates)
		if ok != tt.wantOK || (ok && got != tt.want) {
			t.Errorf("suggest(%q) = %q, %v; want %q, %v", tt.input, got, ok, tt.want, tt.wantOK)
		}
	}
}

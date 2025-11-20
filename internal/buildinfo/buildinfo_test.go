package buildinfo

import (
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	if got, want := Banner(), "msh — My Parallel SSH"; got != want {
		t.Errorf("Banner() = %q, want %q", got, want)
	}
}

func TestStringContainsAllFields(t *testing.T) {
	// Temporarily replace the variables, as -ldflags would, and restore
	// them when the test ends.
	oldV, oldC, oldD := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = oldV, oldC, oldD })
	Version, Commit, Date = "1.2.3", "abc123", "2026-01-02"

	got := String()
	for _, want := range []string{"msh", "1.2.3", "abc123", "2026-01-02"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}

package buildinfo

import (
	"runtime/debug"
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

func TestWithBuildInfo(t *testing.T) {
	base := Info{Version: defaultVersion, Commit: "unknown", Date: "unknown"}
	bi := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.4.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef0123"},
			{Key: "vcs.time", Value: "2026-10-23T10:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	got := withBuildInfo(base, bi)
	if got.Version != "v1.4.0" || got.Commit != "0123456789ab-dirty" || got.Date != "2026-10-23T10:00:00Z" {
		t.Errorf("got %+v", got)
	}

	ldflags := Info{Version: "7.7.7", Commit: "abc", Date: "today"}
	if got := withBuildInfo(ldflags, bi); got != ldflags {
		t.Errorf("-ldflags values must win: %+v", got)
	}
}

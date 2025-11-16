package main

import (
	"strings"
	"testing"
)

// TestBanner checks the exact banner text.
func TestBanner(t *testing.T) {
	want := "msh — My Parallel SSH"

	got := banner()

	if got != want {
		t.Errorf("banner() = %q, want %q", got, want)
	}
}

// TestBannerStartsWithAppName checks that the banner starts with the
// executable's name, which users type on the command line.
func TestBannerStartsWithAppName(t *testing.T) {
	if !strings.HasPrefix(banner(), appName) {
		t.Errorf("banner() = %q, want prefix %q", banner(), appName)
	}
}

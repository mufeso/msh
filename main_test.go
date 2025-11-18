package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	want := "msh — My Parallel SSH"
	if got := banner(); got != want {
		t.Errorf("banner() = %q, want %q", got, want)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if want := "msh version " + version + "\n"; stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{flag}, &stdout, &stderr); code != exitOK {
			t.Fatalf("%s: exit code = %d, want %d", flag, code, exitOK)
		}
		if !strings.Contains(stdout.String(), "Usage:") {
			t.Errorf("%s: stdout = %q, want usage text", flag, stdout.String())
		}
	}
}

func TestRunUnknownArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"bogus"}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "--help") {
		t.Errorf("stderr = %q, want a hint about --help", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

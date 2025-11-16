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

func TestRunWithoutArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(nil, &stdout, &stderr)

	if code != exitOK {
		t.Fatalf("run() exit code = %d, want %d", code, exitOK)
	}
	if got, want := stdout.String(), banner()+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsUnexpectedArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"bogus"}, &stdout, &stderr)

	if code != exitUsage {
		t.Fatalf("run() exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), `"bogus"`) {
		t.Errorf("stderr = %q, want it to mention the argument", stderr.String())
	}
}
